package main

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/lambdawalker/go.infra.composer/reporting"
)

// Exercise the pinned reporting module with this application's manifest and an
// older inventory that lacks dependency evidence. Keep consumer cases here,
// rather than making Composer depend on Card Designer's resource names.
func TestSavedInventoryTopology(t *testing.T) {
	raw, err := os.ReadFile("../deploy.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		ArtifactAttachments map[string]string
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	types := map[string]string{
		"api":             "aws:apigatewayv2/api:Api",
		"lambda":          "aws:apigatewayv2/integration:Integration",
		"api-routes":      "aws:apigatewayv2/route:Route",
		"live":            "aws:apigatewayv2/stage:Stage",
		"users":           "aws:cognito/userPool:UserPool",
		"browser":         "aws:cognito/userPoolClient:UserPoolClient",
		"api-role":        "aws:iam/role:Role",
		"api-permissions": "aws:iam/rolePolicy:RolePolicy",
		"handler":         "aws:lambda/function:Function",
		"gateway":         "aws:lambda/permission:Permission",
		"tokens":          "aws:dynamodb/table:Table",
	}
	report := reporting.Report{}
	for name, typ := range types {
		report.Artifacts = append(report.Artifacts, reporting.Artifact{Key: name, Name: name, Type: typ})
	}
	want := map[string]string{
		"lambda": "api", "api-routes": "api", "live": "api",
		"browser": "users", "api-permissions": "api-role", "gateway": "handler",
	}
	if got := reporting.ResolveAttachments(report.Artifacts, manifest.ArtifactAttachments); !reflect.DeepEqual(got, want) {
		t.Fatalf("saved inventory attachments = %v, want %v", got, want)
	}
	var out bytes.Buffer
	if err := reporting.WriteHTMLWithAttachmentHints(&out, report, manifest.ArtifactAttachments); err != nil {
		t.Fatal(err)
	}
	_, payload, ok := strings.Cut(out.String(), `<script id="attachments" type="application/json">`)
	if !ok {
		t.Fatal("offline report omitted attachment data")
	}
	payload, _, ok = strings.Cut(payload, "</script>")
	var got map[string]string
	if !ok || json.Unmarshal([]byte(payload), &got) != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("offline report lost Card Designer's attachment relationships")
	}
	for _, artifact := range report.Artifacts {
		if artifact.Parent != "" {
			t.Fatalf("presentation changed Pulumi parent of %s", artifact.Name)
		}
	}
}
