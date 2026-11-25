// The consumer adapter only preserves old command aliases and records, builds
// the pinned shared CLI, and delegates. It owns no deployment/cloud workflow.
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
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if e, ok := err.(*exec.ExitError); ok {
			os.Exit(e.ExitCode())
		}
		os.Exit(1)
	}
}
func run(ctx context.Context, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := filepath.Dir(cwd)
	if len(args) > 0 && args[0] == "migrate" {
		return migrateLegacy(root)
	}
	args, err = normalizeArgs(args)
	if err != nil {
		return err
	}
	localOnly := false
	action := ""
	for i, arg := range args {
		if arg == "--help" || arg == "-h" {
			localOnly = true
		}
		if arg == "--action" && i+1 < len(args) {
			action = args[i+1]
		}
		if strings.HasPrefix(arg, "--action=") {
			action = strings.TrimPrefix(arg, "--action=")
		}
	}
	localOnly = localOnly || action == "build" || action == "clean-cache"

	if !localOnly {
		if err = checkLegacy(root); err != nil {
			return err
		}
	}
	dir := filepath.Join(root, ".deploy", "tools")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	// A unique output avoids replacing an executable used by another wizard on Windows.
	temp, err := os.MkdirTemp(dir, "cli-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	name := "infra-deploy"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(temp, name)
	build := exec.CommandContext(ctx, "go", "build", "-mod=readonly", "-o", binary, "github.com/lambdawalker/go.infra.composer/deploycli/cmd/infra-deploy")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err = build.Run(); err != nil {
		return err
	}
	cli := exec.CommandContext(ctx, binary, args...)
	cli.Dir = root
	cli.Stdin = os.Stdin
	cli.Stdout = os.Stdout
	cli.Stderr = os.Stderr
	return cli.Run()
}
func normalizeArgs(args []string) ([]string, error) {
	result := []string{}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		action := args[0]
		args = args[1:]
		switch action {
		case "recover":
			action = "status"
		case "reconfigure":
			action = "setup"
			result = append(result, "--reconfigure")
		case "export":
			return nil, fmt.Errorf("use --export-settings PATH for an allowlisted settings export")
		}
		result = append(result, "--action", action)
	}
	aliases := map[string]string{"--noninteractive": "--non-interactive", "--yes": "--approve", "--state": "--backend", "--certificate-arn": "--certificatearn", "--zone-id": "--zoneid"}
	for i := 0; i < len(args); i++ {
		flag, value, hasValue := strings.Cut(args[i], "=")
		if alias, ok := aliases[flag]; ok {
			flag = alias
		}
		if hasValue {
			if value == "" && (flag == "--domain" || flag == "--certificatearn" || flag == "--zoneid") {
				value = "none"
			}
			result = append(result, flag+"="+value)
			continue
		}
		result = append(result, flag)
		// Preserve literal/empty following values; only optional site inputs use none.
		if (flag == "--domain" || flag == "--certificatearn" || flag == "--zoneid") && i+1 < len(args) && args[i+1] == "" {
			result = append(result, "none")
			i++
		}
	}
	return result, nil
}
