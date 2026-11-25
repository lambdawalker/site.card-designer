package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

type legacySettings struct {
	Version                                                                               int
	Project, Environment, Account, Region, Profile, State, Domain, CertificateARN, ZoneID string
	Initialized                                                                           bool
}

func readJSON(path string, v any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}
func legacyProject(root string) (string, error) {
	var identity struct{ Project string }
	err := readJSON(filepath.Join(root, ".deploy/identity.json"), &identity)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9-]{1,39}$`).MatchString(identity.Project) {
		return "", fmt.Errorf("invalid legacy project identity")
	}
	return identity.Project, nil
}
func checkLegacy(root string) error {
	project, err := legacyProject(root)
	if err != nil || project == "" {
		return err
	}
	var receipt struct {
		Version       int
		Project, Root string
	}
	if err = readJSON(filepath.Join(root, ".deploy/legacy-migration.json"), &receipt); err == nil && receipt.Version == 1 && receipt.Project == project && receipt.Root == root {
		return nil
	}
	return fmt.Errorf("legacy deployment records found; run deploy.sh migrate (Windows: deploy.bat migrate), then reuse the original Pulumi passphrase; see deploy/README.md")
}

// This is a nonsecret, offline record conversion. It copies no cloud success or
// new ownership. Existing initialized-stack receipts prevent silent recreation.
// Original settings and encrypted vaults remain byte-for-byte intact.
func migrateLegacy(root string) error {
	project, err := legacyProject(root)
	if err != nil {
		return err
	}
	if project == "" {
		return fmt.Errorf("no legacy .deploy/identity.json found")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	central := filepath.Join(home, ".infra-composer", project)
	entries, err := os.ReadDir(central)
	if err != nil {
		return err
	}
	records := map[string]any{}
	identity := struct {
		Version    int
		Name, Root string
	}{1, project, root}
	records[filepath.Join(central, "project.json")] = identity
	records[filepath.Join(root, ".deploy/project.json")] = identity
	environments := []string{}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "environments" {
			continue
		}
		path := filepath.Join(central, entry.Name(), "settings.json")
		var old legacySettings
		if err = readJSON(path, &old); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		if old.Version != 1 || old.Project != project || old.Environment != entry.Name() || !regexp.MustCompile(`^[a-z][a-z0-9-]{0,30}$`).MatchString(old.Environment) || !regexp.MustCompile(`^\d{12}$`).MatchString(old.Account) || !regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-\d$`).MatchString(old.Region) || !regexp.MustCompile(`^s3://[a-z0-9.-]+/[^?\s]+$`).MatchString(old.State) {
			return fmt.Errorf("invalid/incomplete legacy scope in %s; recover original settings before migration", path)
		}
		target := filepath.Join(central, "environments", old.Environment)
		records[filepath.Join(target, "settings.json")] = struct {
			Version                                        int
			Environment, Profile, Region, Account, Backend string
		}{1, old.Environment, old.Profile, old.Region, old.Account, old.State}
		records[filepath.Join(target, "stack.json")] = struct {
			Version                         int
			Account, Region, Backend, Stack string
			Initialized                     bool
		}{1, old.Account, old.Region, old.State, project + "-" + old.Environment, old.Initialized}
		optional := func(s string) string {
			if s == "" {
				return "none"
			}
			return s
		}
		records[filepath.Join(target, "designer.json")] = struct {
			Version                        int
			Domain, CertificateARN, ZoneID string
		}{1, optional(old.Domain), optional(old.CertificateARN), optional(old.ZoneID)}
		environments = append(environments, old.Environment)
	}
	if len(environments) == 0 {
		return fmt.Errorf("no complete legacy settings found under %s", central)
	}
	// Validate every destination before changing any record. Exact existing bytes
	// are accepted for restart after interruption; differing records are preserved.
	encoded := map[string][]byte{}
	for path, value := range records {
		data, e := json.MarshalIndent(value, "", "  ")
		if e != nil {
			return e
		}
		encoded[path] = data
		if old, e := os.ReadFile(path); e == nil && string(old) != string(data) {
			return fmt.Errorf("migration destination differs: %s; existing records preserved", path)
		} else if e != nil && !os.IsNotExist(e) {
			return e
		}
	}
	for path, data := range encoded {
		if err = privateWrite(path, data); err != nil {
			return err
		}
	}
	receipt, _ := json.MarshalIndent(struct {
		Version       int
		Project, Root string
		Environments  []string
	}{1, project, root, environments}, "", "  ")
	if err = privateWrite(filepath.Join(root, ".deploy/legacy-migration.json"), receipt); err != nil {
		return err
	}
	fmt.Println("Copied legacy settings and stack identity for", project, environments)
	fmt.Println("Original records and vaults preserved. Reuse the ORIGINAL Pulumi passphrase on first Composer unlock.")
	fmt.Println("Existing stacks require interactive --adopt-legacy-stack review. No AWS resources were read or changed.")
	return nil
}
func privateWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".migration-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(data)
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	if existing, e := os.ReadFile(path); e == nil && string(existing) == string(data) {
		return nil
	}
	return os.Rename(f.Name(), path)
}
