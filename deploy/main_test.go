package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCommandCompatibility(t *testing.T) {
	for _, tc := range []struct{ in, out []string }{
		{[]string{"preview", "--state", "s3://bucket/prefix", "--domain", ""}, []string{"--action", "preview", "--backend", "s3://bucket/prefix", "--domain", "none"}},
		{[]string{"deploy", "--noninteractive", "--yes", "--certificate-arn=", "--zone-id", ""}, []string{"--action", "deploy", "--non-interactive", "--approve", "--certificatearn=none", "--zoneid", "none"}},
		{[]string{"recover"}, []string{"--action", "status"}},
		{[]string{"--server", "--no-open"}, []string{"--server", "--no-open"}},
	} {
		got, err := normalizeArgs(tc.in)
		if err != nil || !reflect.DeepEqual(got, tc.out) {
			t.Fatalf("%v: %v %v", tc.in, got, err)
		}
	}
}
func TestLegacyMigrationPreservesScopeAndVault(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	os.MkdirAll(filepath.Join(root, ".deploy/dev"), 0700)
	os.WriteFile(filepath.Join(root, ".deploy/identity.json"), []byte(`{"Project":"designer"}`), 0600)
	vault := []byte("original encrypted vault bytes")
	os.WriteFile(filepath.Join(root, ".deploy/dev/vault.json"), vault, 0600)
	dir := filepath.Join(home, ".infra-composer/designer/dev")
	os.MkdirAll(dir, 0700)
	old := legacySettings{Version: 1, Project: "designer", Environment: "dev", Account: "123456789012", Region: "us-east-1", Profile: "sso", State: "s3://existing/original", Initialized: true}
	raw, _ := json.Marshal(old)
	os.WriteFile(filepath.Join(dir, "settings.json"), raw, 0600)
	if checkLegacy(root) == nil {
		t.Fatal("legacy use was not blocked")
	}
	if err := migrateLegacy(root); err != nil {
		t.Fatal(err)
	}
	if err := checkLegacy(root); err != nil {
		t.Fatal(err)
	}
	if err := migrateLegacy(root); err != nil {
		t.Fatal("migration not restartable", err)
	}
	var receipt struct {
		Stack, Backend string
		Initialized    bool
	}
	if err := readJSON(filepath.Join(home, ".infra-composer/designer/environments/dev/stack.json"), &receipt); err != nil {
		t.Fatal(err)
	}
	if !receipt.Initialized || receipt.Stack != "designer-dev" || receipt.Backend != old.State {
		t.Fatal(receipt)
	}
	got, _ := os.ReadFile(filepath.Join(root, ".deploy/dev/vault.json"))
	if string(got) != string(vault) {
		t.Fatal("vault changed")
	}
	got, _ = os.ReadFile(filepath.Join(dir, "settings.json"))
	if string(got) != string(raw) {
		t.Fatal("original settings changed")
	}
	// Changed target settings must stop, even if a prior migration was complete.
	os.WriteFile(filepath.Join(home, ".infra-composer/designer/environments/dev/settings.json"), []byte("different target"), 0600)
	if migrateLegacy(root) == nil {
		t.Fatal("conflicting target overwritten")
	}
}
func TestMigrationRejectsIncompleteRecords(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	os.MkdirAll(filepath.Join(root, ".deploy"), 0700)
	os.WriteFile(filepath.Join(root, ".deploy/identity.json"), []byte(`{"Project":"designer"}`), 0600)
	dir := filepath.Join(home, ".infra-composer/designer/dev")
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"Version":1,"Project":"designer","Environment":"dev"}`), 0600)
	if migrateLegacy(root) == nil {
		t.Fatal("incomplete migration accepted")
	}
	if _, err := os.Stat(filepath.Join(root, ".deploy/project.json")); !os.IsNotExist(err) {
		t.Fatal("published identity before validation")
	}
}
