package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type cloudMocks struct {
	sync.Mutex
	inputs map[string]resource.PropertyMap
}

func (m *cloudMocks) NewResource(a pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	m.Lock()
	defer m.Unlock()
	m.inputs[a.Name] = a.Inputs
	out := a.Inputs.Copy()
	for k, v := range map[string]string{"arn": "arn:aws:mock::123456789012:" + a.Name, "name": a.Name, "bucket": a.Name, "domainName": "example.cloudfront.net", "bucketRegionalDomainName": "example.s3.amazonaws.com", "apiEndpoint": "https://example.execute-api.us-east-1.amazonaws.com", "executionArn": "arn:aws:execute-api:us-east-1:123456789012:example", "invokeArn": "arn:aws:lambda:us-east-1:123456789012:function:example", "hostedZoneId": "ZTEST"} {
		if _, ok := out[resource.PropertyKey(k)]; !ok {
			out[resource.PropertyKey(k)] = resource.NewStringProperty(v)
		}
	}
	return a.Name + "-id", out, nil
}
func (m *cloudMocks) Call(a pulumi.MockCallArgs) (resource.PropertyMap, error) {
	if strings.Contains(a.Token, "getRegion") {
		return resource.NewPropertyMapFromMap(map[string]interface{}{"name": "us-east-1", "description": "test", "endpoint": "amazonaws.com"}), nil
	}
	return resource.NewPropertyMapFromMap(map[string]interface{}{"accountId": "123456789012", "arn": "arn:aws:iam::123456789012:role/test", "userId": "test"}), nil
}
func TestInfrastructureAuthAndCacheContracts(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{"infra", "backend", "web/dist"} {
		if e := os.MkdirAll(filepath.Join(dir, p), 0700); e != nil {
			t.Fatal(e)
		}
	}
	f, e := os.Create(filepath.Join(dir, "backend/bootstrap.zip"))
	if e != nil {
		t.Fatal(e)
	}
	z := zip.NewWriter(f)
	entry, _ := z.Create("bootstrap")
	entry.Write([]byte("test"))
	z.Close()
	f.Close()
	os.WriteFile(filepath.Join(dir, "web/dist/index.html"), []byte("test"), 0600)
	t.Chdir(filepath.Join(dir, "infra"))
	m := &cloudMocks{inputs: map[string]resource.PropertyMap{}}
	e = pulumi.RunErr(run, pulumi.WithMocks("card-designer", "test", m), func(i *pulumi.RunInfo) {
		i.Config = map[string]string{"card-designer:project": "designer", "card-designer:environment": "test"}
	})
	if e != nil {
		t.Fatal(e)
	}
	table := m.inputs["tokens"]
	if table["hashKey"].StringValue() != "token_hash" {
		t.Fatal("token lookup key mismatch")
	}
	index := table["globalSecondaryIndexes"].ArrayValue()[0].ObjectValue()
	if index["name"].StringValue() != "owner-index" || index["hashKey"].StringValue() != "owner" {
		t.Fatal("token owner index mismatch")
	}
	client := m.inputs["browser"]
	if client["generateSecret"].BoolValue() {
		t.Fatal("browser must use a public PKCE client")
	}
	if client["allowedOauthFlows"].ArrayValue()[0].StringValue() != "code" {
		t.Fatal("only authorization code flow allowed")
	}
	dist := m.inputs["site"]
	foundAPI, foundWorker := false, false
	for _, b := range dist["orderedCacheBehaviors"].ArrayValue() {
		v := b.ObjectValue()
		switch v["pathPattern"].StringValue() {
		case "/api/*":
			foundAPI = true
			if v["cachePolicyId"].StringValue() != "4135ea2d-6df8-44a3-9df3-4b5a84be39ad" || v["originRequestPolicyId"].StringValue() != "b689b0a8-53d0-40ab-baf2-68738e2966ac" {
				t.Fatal("API caching/Authorization forwarding mismatch")
			}
		case "/assets/template-worker-*":
			foundWorker = true
			if v["responseHeadersPolicyId"].StringValue() == "" {
				t.Fatal("missing worker CSP")
			}
		}
	}
	if !foundAPI || !foundWorker {
		t.Fatal("missing protected CloudFront behaviors")
	}
	for _, key := range []string{"web-private", "data-private"} {
		p := m.inputs[key]
		for _, field := range []resource.PropertyKey{"blockPublicAcls", "blockPublicPolicy", "ignorePublicAcls", "restrictPublicBuckets"} {
			if !p[field].BoolValue() {
				t.Fatalf("%s lacks %s", key, field)
			}
		}
	}
	env := m.inputs["handler"]["environment"].ObjectValue()["variables"].ObjectValue()
	for _, key := range []resource.PropertyKey{"DATA_BUCKET", "TOKEN_TABLE", "COGNITO_CLIENT_ID", "COGNITO_USER_POOL_ID", "COGNITO_DOMAIN", "REDIRECT_URI", "LOGOUT_URI"} {
		if env[key].StringValue() == "" {
			t.Fatalf("missing runtime config %s", key)
		}
	}
}
