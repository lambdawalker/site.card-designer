// assets prepares the website and shared catalog. Infrastructure and approvals
// belong to Infra Composer; this helper has no AWS/Pulumi responsibilities.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := build(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func build(ctx context.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := filepath.Dir(cwd)
	node, err := exec.CommandContext(ctx, "node", "--version").Output()
	if err != nil {
		return fmt.Errorf("install Node 24 and npm: %w", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(node)), "v24.") {
		return fmt.Errorf("Node 24 is required; found %s", strings.TrimSpace(string(node)))
	}
	npm := "npm"
	if runtime.GOOS == "windows" {
		npm = "npm.cmd"
	}
	run := func(dir string, args ...string) error {
		commandArgs := args
		name := npm
		if runtime.GOOS == "windows" {
			name = "cmd.exe"
			commandArgs = append([]string{"/d", "/c", "npm.cmd"}, args...)
		}
		c := exec.CommandContext(ctx, name, commandArgs...)
		c.Dir = filepath.Join(root, dir)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}
	if err = run("packages/drawing-tools", "ci"); err != nil {
		return err
	}
	if err = run("web", "ci"); err != nil {
		return err
	}
	if err = run("web", "run", "build"); err != nil {
		return err
	}
	// Copy only after the frontend succeeded; failed preparations publish no cache.
	source, err := os.ReadFile(filepath.Join(root, "assets/catalog.json"))
	if err != nil {
		return err
	}
	target := filepath.Join(root, "backend/internal/catalog/catalog.json")
	old, err := os.ReadFile(target)
	if err == nil && string(old) == string(source) {
		return nil
	}
	return os.WriteFile(target, source, 0644)
}
