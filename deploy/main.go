package main

import (
	"archive/zip"

	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"golang.org/x/crypto/scrypt"
	"golang.org/x/term"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Settings struct {
	Version                                                                               int
	Project, Environment, Account, Region, Profile, State, Domain, CertificateARN, ZoneID string
	Initialized                                                                           bool
}
type Vault struct {
	Version           int
	Salt, Nonce, Data []byte
}

var root string
var safe = regexp.MustCompile(`^[a-z][a-z0-9-]{1,39}$`)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "Deployment stopped:", e)
		os.Exit(1)
	}
}
func command(dir string, env []string, name string, args ...string) ([]byte, error) {
	c := exec.Command(name, args...)
	c.Dir = dir
	c.Env = append(os.Environ(), env...)
	var b bytes.Buffer
	c.Stdout = io.MultiWriter(os.Stdout, &b)
	c.Stderr = os.Stderr
	e := c.Run()
	return b.Bytes(), e
}
func quiet(dir string, env []string, name string, args ...string) ([]byte, error) {
	c := exec.Command(name, args...)
	c.Dir = dir
	c.Env = append(os.Environ(), env...)
	b, e := c.Output()
	if e != nil {
		return nil, fmt.Errorf("%s failed: %w", name, e)
	}
	return b, nil
}
func atomic(path string, b []byte) error {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".pending-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(b)
	}
	if ce := f.Close(); e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}
func save(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return atomic(path, b)
}
func load(path string, v any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}
func encrypt(scope string, password, data []byte) (Vault, error) {
	v := Vault{Version: 1, Salt: make([]byte, 16), Nonce: make([]byte, 12)}
	rand.Read(v.Salt)
	rand.Read(v.Nonce)
	k, e := scrypt.Key(password, v.Salt, 32768, 8, 1, 32)
	if e != nil {
		return v, e
	}
	b, e := aes.NewCipher(k)
	if e != nil {
		return v, e
	}
	a, e := cipher.NewGCM(b)
	if e != nil {
		return v, e
	}
	v.Data = a.Seal(nil, v.Nonce, data, []byte(scope))
	return v, nil
}
func decrypt(scope string, password []byte, v Vault) ([]byte, error) {
	if v.Version != 1 || len(v.Salt) != 16 || len(v.Nonce) != 12 {
		return nil, errors.New("invalid vault")
	}
	k, e := scrypt.Key(password, v.Salt, 32768, 8, 1, 32)
	if e != nil {
		return nil, e
	}
	b, _ := aes.NewCipher(k)
	a, _ := cipher.NewGCM(b)
	return a.Open(nil, v.Nonce, v.Data, []byte(scope))
}
func secret(label string) ([]byte, error) {
	fmt.Fprint(os.Stderr, label+": ")
	b, e := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if e == nil && len(b) == 0 {
		e = errors.New("secret cannot be empty")
	}
	return b, e
}

// A raw terminal reader has a bounded timeout and owns no background goroutine.
func prompt(label, def string, timed bool) (string, error) {
	fmt.Fprintf(os.Stderr, "%s [%s]", label, def)
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", errors.New("interactive input requires a terminal; use explicit flags")
	}
	state, e := term.MakeRaw(int(os.Stdin.Fd()))
	if e != nil {
		return "", e
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	var answer []byte
	deadline := time.Now().Add(5 * time.Second)
	for {
		if timed {
			fmt.Fprintf(os.Stderr, "\r%s [%s] Continuing in %d seconds; Enter accepts, Esc cancels. ", label, def, int(time.Until(deadline).Seconds())+1)
		}
		b, e := readKey(os.Stdin, 100*time.Millisecond)
		if e != nil {
			return "", e
		}
		if b != 0 {
			timed = false
			if (b == 'n' || b == 'N') && strings.Contains(label, "Y/n") {
				fmt.Fprint(os.Stderr, "\r\n")
				return "n", nil
			}
			switch b {
			case 3, 27:
				return "", errors.New("cancelled")
			case 13, 10:
				fmt.Fprint(os.Stderr, "\r\n")
				if len(answer) == 0 {
					return def, nil
				}
				return string(answer), nil
			case 127, 8:
				if len(answer) > 0 {
					answer = answer[:len(answer)-1]
				}
			default:
				answer = append(answer, b)
				fmt.Fprintf(os.Stderr, "%c", b)
			}
		}
		if timeoutAcceptsDefault(timed, len(answer), time.Now(), deadline) {
			fmt.Fprint(os.Stderr, "\r\n")
			return def, nil
		}
	}
}
func run() error {
	cwd, _ := os.Getwd()
	root = filepath.Dir(cwd)
	action := "deploy"
	args := os.Args[1:]
	validActions := map[string]bool{"deploy": true, "preview": true, "setup": true, "build": true, "status": true, "recover": true, "report": true, "export": true, "reconfigure": true}
	explicit := false
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		action = args[0]
		args = args[1:]
		explicit = true
	}
	if !validActions[action] {
		return errors.New("unknown action")
	}
	f := flag.NewFlagSet("deploy", flag.ContinueOnError)
	s := Settings{Version: 1}
	f.StringVar(&s.Project, "project", "", "project identity")
	f.StringVar(&s.Environment, "environment", "", "environment")
	f.StringVar(&s.Region, "region", "", "AWS region")
	f.StringVar(&s.Account, "account", "", "expected AWS account")
	f.StringVar(&s.Profile, "profile", "", "SSO profile")
	f.StringVar(&s.State, "state", "", "s3://bucket/project prefix")
	f.StringVar(&s.Domain, "domain", "", "optional hostname")
	f.StringVar(&s.CertificateARN, "certificate-arn", "", "issued us-east-1 ACM certificate")
	f.StringVar(&s.ZoneID, "zone-id", "", "Route53 zone, empty means manual DNS")
	ci := f.Bool("noninteractive", false, "require explicit inputs")
	yes := f.Bool("yes", false, "approve explicit deploy in CI")
	if e := f.Parse(args); e != nil {
		return e
	}
	if action == "build" {
		return build()
	}
	if *ci && action == "deploy" && (!explicit || !*yes) {
		return errors.New("CI deploy requires explicit deploy --noninteractive --yes")
	}
	for _, tool := range []string{"aws", "pulumi", "go", "npm", "node"} {
		if _, e := exec.LookPath(tool); e != nil {
			return fmt.Errorf("install required tool %s and add it to PATH", tool)
		}
	}
	fmt.Println("Card Designer deployment")
	var identity struct{ Project string }
	_ = load(filepath.Join(root, ".deploy", "identity.json"), &identity)
	if s.Project == "" {
		s.Project = identity.Project
	}
	if s.Project == "" {
		if *ci {
			return errors.New("--project required")
		}
		v, e := prompt("Project name", "card-designer", true)
		if e != nil {
			return e
		}
		s.Project = v
	}
	if !safe.MatchString(s.Project) {
		return errors.New("invalid project name")
	}
	if s.Environment == "" {
		if *ci {
			return errors.New("--environment required")
		}
		v, e := prompt("Environment", "dev", true)
		if e != nil {
			return e
		}
		s.Environment = v
	}
	if !safe.MatchString(s.Environment) {
		return errors.New("invalid environment")
	}
	if err := validateIdentity(s.Project, s.Environment); err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	record := filepath.Join(home, ".infra-composer", s.Project, s.Environment, "settings.json")
	old := Settings{}
	if e := load(record, &old); e == nil {
		if old.Project != s.Project || old.Environment != s.Environment {
			return errors.New("settings scope mismatch")
		}
		if action != "reconfigure" {
			merge(&s, old)
		} else {
			s.Initialized = old.Initialized
			if s.State == "" {
				s.State = old.State
			}
			if s.Profile == "" {
				s.Profile = old.Profile
			}
		}
		f.Visit(func(fl *flag.Flag) {
			if fl.Name == "domain" {
				s.Domain = fl.Value.String()
			}
			if fl.Name == "zone-id" {
				s.ZoneID = fl.Value.String()
			}
			if fl.Name == "certificate-arn" {
				s.CertificateARN = fl.Value.String()
			}
		})
		fmt.Println("Configuration: saved settings plus explicit flags")
	} else if !os.IsNotExist(e) {
		return e
	}
	if s.Domain == "" && !*ci && !s.Initialized {
		v, e := prompt("Site hostname (Enter for generated CloudFront URL)", "", false)
		if e != nil {
			return e
		}
		s.Domain = v
		if v != "" && s.CertificateARN == "" {
			v, e = prompt("Issued ACM certificate ARN in us-east-1", "", false)
			if e != nil {
				return e
			}
			s.CertificateARN = v
		}
	}
	if s.Region == "" {
		if *ci {
			return errors.New("--region required")
		}
		v, e := prompt("AWS region", "us-east-1", true)
		if e != nil {
			return e
		}
		s.Region = v
	}
	if !*ci && s.Profile == "" {
		b, e := quiet(root, nil, "aws", "configure", "list-profiles")
		if e != nil {
			return e
		}
		profiles := strings.Fields(string(b))
		if len(profiles) == 0 {
			return errors.New("configure an AWS SSO profile with aws configure sso")
		}
		v, e := prompt("AWS SSO profile", profiles[0], true)
		if e != nil {
			return e
		}
		s.Profile = v
	}
	env := []string{"AWS_REGION=" + s.Region, "AWS_DEFAULT_REGION=" + s.Region}
	if s.Profile != "" {
		env = append(env, "AWS_PROFILE="+s.Profile)
	}
	if !*ci {
		validSSO := false
		for _, key := range []string{"sso_session", "sso_start_url"} {
			b, e := quiet(root, env, "aws", "configure", "get", key, "--profile", s.Profile)
			if e == nil && strings.TrimSpace(string(b)) != "" {
				validSSO = true
			}
		}
		if !validSSO {
			return errors.New("selected profile is not IAM Identity Center; run aws configure sso")
		}
	}
	id, e := quiet(root, env, "aws", "sts", "get-caller-identity", "--output", "json")
	if e != nil && !*ci {
		if _, e = command(root, env, "aws", "sso", "login", "--profile", s.Profile); e == nil {
			id, e = quiet(root, env, "aws", "sts", "get-caller-identity", "--output", "json")
		}
	}
	if e != nil {
		return e
	}
	var caller struct{ Account, Arn string }
	if e = json.Unmarshal(id, &caller); e != nil {
		return e
	}
	if s.Account != "" && s.Account != caller.Account {
		return errors.New("AWS account mismatch; use deliberate reconfiguration")
	}
	s.Account = caller.Account
	fmt.Printf("\nTarget\n  Project      %s\n  Environment  %s\n  Account      %s\n  Identity     %s\n  Region       %s\n  Domain       %s\n", s.Project, s.Environment, s.Account, caller.Arn, s.Region, s.Domain)
	if !*ci {
		v, e := prompt("Proceed with this target? Y/n", "Y", true)
		if e != nil {
			return e
		}
		if strings.ToLower(v) != "y" && strings.ToLower(v) != "yes" {
			return errors.New("cancelled")
		}
	}
	if s.State == "" {
		if *ci {
			return errors.New("--state required; CI never bootstraps")
		}
		v, e := prompt("Do you have an S3 bucket for deployment state? y/N (No creates a private encrypted versioned bucket)", "N", true)
		if e != nil {
			return e
		}
		if strings.ToLower(v) == "y" {
			v, e = prompt("Existing state bucket", "", false)
			if e != nil {
				return e
			}
			s.State = "s3://" + v + "/" + s.Project
		} else {
			suffix := make([]byte, 4)
			rand.Read(suffix)
			bucket := fmt.Sprintf("%s-state-%s-%x", s.Project, s.Account, suffix)
			s.State = "s3://" + bucket + "/" + s.Project
			if e = save(record, s); e != nil {
				return e
			}
			if e = bootstrap(s, env, bucket); e != nil {
				return e
			}
		}
	}
	if !strings.HasPrefix(s.State, "s3://") || !strings.Contains(strings.TrimPrefix(s.State, "s3://"), "/") {
		return errors.New("state must be s3://bucket/dedicated-prefix")
	}
	if e = verifyState(s, env); e != nil {
		return e
	}
	if e = save(record, s); e != nil {
		return e
	}
	if e = save(filepath.Join(root, ".deploy", "identity.json"), map[string]string{"Project": s.Project}); e != nil {
		return e
	}
	if action == "export" {
		return save(filepath.Join(root, ".deploy", "export.json"), s)
	}
	pass := os.Getenv("PULUMI_CONFIG_PASSPHRASE")
	if pass == "" {
		if *ci {
			return errors.New("PULUMI_CONFIG_PASSPHRASE required")
		}
		scope := s.Project + "/" + s.Environment + "/" + s.Account + "/" + s.Region
		vaultPath := filepath.Join(root, ".deploy", s.Environment, "vault.json")
		var vault Vault
		e = load(vaultPath, &vault)
		pw, pe := secret("Local vault unlock password")
		if pe != nil {
			return pe
		}
		if e == nil {
			b, e := decrypt(scope, pw, vault)
			if e != nil {
				return errors.New("vault unlock failed; original left unchanged")
			}
			pass = string(b)
		} else if os.IsNotExist(e) {
			confirm, e := secret("Confirm local vault password")
			if e != nil {
				return e
			}
			if !bytes.Equal(pw, confirm) {
				return errors.New("passwords differ")
			}
			p, e := secret("Pulumi passphrase (existing stack: use its original passphrase)")
			if e != nil {
				return e
			}
			q, e := secret("Confirm Pulumi passphrase")
			if e != nil || !bytes.Equal(p, q) {
				return errors.New("passphrases differ")
			}
			vault, e = encrypt(scope, pw, p)
			if e != nil {
				return e
			}
			if e = save(vaultPath, vault); e != nil {
				return e
			}
			pass = string(p)
		} else {
			return e
		}
	}
	env = append(env, "PULUMI_CONFIG_PASSPHRASE="+pass)
	infra := filepath.Join(root, "infra")
	if _, e = command(infra, env, "pulumi", "login", s.State); e != nil {
		return e
	}
	stack := s.Project + "-" + s.Environment
	b, e := quiet(infra, env, "pulumi", "stack", "ls", "--json")
	if e != nil {
		return e
	}
	var stacks []struct{ Name string }
	if e = json.Unmarshal(b, &stacks); e != nil {
		return e
	}
	found := false
	for _, st := range stacks {
		if st.Name == stack {
			found = true
		}
	}
	if !found {
		if s.Initialized || action == "status" || action == "recover" || action == "report" {
			return errors.New("initialized stack missing; restore state backup, do not reinitialize")
		}
		if _, e = command(infra, env, "pulumi", "stack", "init", stack, "--secrets-provider", "passphrase"); e != nil {
			return e
		}
		s.Initialized = true
		if e = save(record, s); e != nil {
			return e
		}
	} else {
		if _, e = command(infra, env, "pulumi", "stack", "select", stack); e != nil {
			return e
		}
		if _, e = command(infra, env, "pulumi", "config", "refresh", "--force"); e != nil {
			return fmt.Errorf("restore authoritative encrypted config: %w", e)
		}
	}
	if action == "status" || action == "recover" || action == "report" {
		_, e = command(infra, env, "pulumi", "stack", "output", "--json")
		return e
	}
	for _, kv := range [][2]string{{"aws:region", s.Region}, {"project", s.Project}, {"environment", s.Environment}, {"domain", s.Domain}, {"certificateArn", s.CertificateARN}, {"zoneId", s.ZoneID}} {
		if _, e = command(infra, env, "pulumi", "config", "set", kv[0], kv[1]); e != nil {
			return e
		}
	}
	if action == "setup" || action == "reconfigure" {
		return nil
	}
	if action != "preview" && action != "deploy" {
		return errors.New("unknown action")
	}
	if e = build(); e != nil {
		return e
	}
	start := time.Now()
	plan := filepath.Join(root, ".deploy", s.Environment, "plan.json")
	if e = os.MkdirAll(filepath.Dir(plan), 0700); e != nil {
		return e
	}
	_, e = command(infra, env, "pulumi", "preview", "--refresh", "--save-plan", plan)
	report(s, action, start, e)
	if e != nil {
		return e
	}
	if action == "preview" {
		return nil
	}
	if !*ci {
		v, e := prompt("Deploy the previewed changes? Y/n; press n or Ctrl+C to cancel", "Y", true)
		if e != nil {
			return e
		}
		if strings.ToLower(v) != "y" && strings.ToLower(v) != "yes" {
			return errors.New("cancelled")
		}
	}
	_, e = command(infra, env, "pulumi", "up", "--yes", "--plan", plan, "--skip-preview")
	report(s, "deploy", start, e)
	if e == nil {
		b, oe := quiet(infra, env, "pulumi", "stack", "output", "siteUrl")
		if oe != nil {
			return oe
		}
		client := http.Client{Timeout: 15 * time.Second}
		resp, he := client.Get(strings.TrimSpace(string(b)) + "/api/config")
		if he != nil {
			return fmt.Errorf("applied but health check failed: %w", he)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("applied but API config returned %d", resp.StatusCode)
		}
	}
	return e
}
func merge(s *Settings, o Settings) {
	if s.Region == "" {
		s.Region = o.Region
	}
	if s.Account == "" {
		s.Account = o.Account
	}
	if s.Profile == "" {
		s.Profile = o.Profile
	}
	if s.State == "" {
		s.State = o.State
	}
	if s.Domain == "" {
		s.Domain = o.Domain
	}
	if s.CertificateARN == "" {
		s.CertificateARN = o.CertificateARN
	}
	if s.ZoneID == "" {
		s.ZoneID = o.ZoneID
	}
	s.Initialized = o.Initialized
}
func build() error {
	fmt.Println("Build: Lambda and frontend")
	fingerprint, e := buildFingerprint()
	if e != nil {
		return e
	}
	cachePath := filepath.Join(root, ".deploy", "build-cache.json")
	var cache buildCache
	if load(cachePath, &cache) == nil && cache.Input == fingerprint {
		artifact, e := buildArtifacts()
		if e == nil && artifact == cache.Artifact {
			fmt.Println("Build: verified cached artifacts")
			return nil
		}
	}
	if _, e := command(filepath.Join(root, "packages", "drawing-tools"), nil, "npm", "ci"); e != nil {
		return e
	}
	catalog, e := os.ReadFile(filepath.Join(root, "assets", "catalog.json"))
	if e != nil {
		return e
	}
	if e = atomic(filepath.Join(root, "backend", "internal", "catalog", "catalog.json"), catalog); e != nil {
		return e
	}
	if _, e = command(filepath.Join(root, "backend"), []string{"GOOS=linux", "GOARCH=arm64", "CGO_ENABLED=0"}, "go", "build", "-trimpath", "-ldflags=-s -w", "-o", "bootstrap", "./cmd/lambda"); e != nil {
		return e
	}
	binary, e := os.ReadFile(filepath.Join(root, "backend", "bootstrap"))
	if e != nil {
		return e
	}
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	h := &zip.FileHeader{Name: "bootstrap", Method: zip.Deflate}
	h.SetMode(0755)
	z, e := w.CreateHeader(h)
	if e != nil {
		return e
	}
	z.Write(binary)
	if e = w.Close(); e != nil {
		return e
	}
	if e = atomic(filepath.Join(root, "backend", "bootstrap.zip"), b.Bytes()); e != nil {
		return e
	}
	if _, e = command(filepath.Join(root, "web"), nil, "npm", "ci"); e != nil {
		return e
	}
	_, e = command(filepath.Join(root, "web"), nil, "npm", "run", "build")
	if e != nil {
		return e
	}
	artifact, e := buildArtifacts()
	if e != nil {
		return e
	}
	return save(cachePath, buildCache{fingerprint, artifact})
}
func bootstrap(s Settings, env []string, bucket string) error {
	args := []string{"s3api", "create-bucket", "--bucket", bucket}
	if s.Region != "us-east-1" {
		args = append(args, "--create-bucket-configuration", "LocationConstraint="+s.Region)
	}
	if _, e := command(root, env, "aws", args...); e != nil {
		return e
	}
	for _, args := range [][]string{{"s3api", "put-bucket-versioning", "--bucket", bucket, "--versioning-configuration", "Status=Enabled"}, {"s3api", "put-bucket-encryption", "--bucket", bucket, "--server-side-encryption-configuration", `{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}`}, {"s3api", "put-public-access-block", "--bucket", bucket, "--public-access-block-configuration", "BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true"}, {"s3api", "put-bucket-policy", "--bucket", bucket, "--policy", fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Deny","Principal":"*","Action":"s3:*","Resource":["arn:aws:s3:::%s","arn:aws:s3:::%s/*"],"Condition":{"Bool":{"aws:SecureTransport":"false"}}}]}`, bucket, bucket)}} {
		if _, e := command(root, env, "aws", args...); e != nil {
			return fmt.Errorf("bootstrap incomplete; harden saved bucket before retry: %w", e)
		}
	}
	return nil
}
func verifyState(s Settings, env []string) error {
	bucket := strings.Split(strings.TrimPrefix(s.State, "s3://"), "/")[0]
	if _, e := quiet(root, env, "aws", "s3api", "head-bucket", "--bucket", bucket, "--expected-bucket-owner", s.Account); e != nil {
		return e
	}
	for _, op := range []string{"get-bucket-versioning", "get-bucket-encryption", "get-public-access-block", "get-bucket-location", "get-bucket-policy"} {
		b, e := quiet(root, env, "aws", "s3api", op, "--bucket", bucket, "--expected-bucket-owner", s.Account)
		if e != nil {
			return e
		}
		text := string(b)
		switch op {
		case "get-bucket-versioning":
			if !strings.Contains(text, "Enabled") {
				return errors.New("state versioning required")
			}
		case "get-public-access-block":
			if strings.Count(text, "true") != 4 {
				return errors.New("state public access blocking required")
			}
		case "get-bucket-encryption":
			var doc struct {
				Rules []struct{ ApplyServerSideEncryptionByDefault struct{ SSEAlgorithm string } }
			}
			if json.Unmarshal(b, &doc) != nil || len(doc.Rules) == 0 {
				return errors.New("state encryption missing")
			}
			for _, r := range doc.Rules {
				if r.ApplyServerSideEncryptionByDefault.SSEAlgorithm != "AES256" && r.ApplyServerSideEncryptionByDefault.SSEAlgorithm != "aws:kms" {
					return errors.New("state encryption algorithm invalid")
				}
			}
		case "get-bucket-policy":
			var outer struct{ Policy string }
			if json.Unmarshal(b, &outer) != nil || !tlsPolicy(outer.Policy, bucket) {
				return errors.New("state TLS deny policy required")
			}
		case "get-bucket-location":
			var r struct{ LocationConstraint *string }
			json.Unmarshal(b, &r)
			region := "us-east-1"
			if r.LocationConstraint != nil {
				region = *r.LocationConstraint
			}
			if region != s.Region {
				return errors.New("state bucket region mismatch")
			}
		}
	}
	return nil
}
func report(s Settings, operation string, start time.Time, err error) {
	outcome := "success"
	if err != nil {
		outcome = "failed"
	}
	snapshot := map[string]any{"schema": 1, "owner": "lambdawalker/site.card-designer", "project": s.Project, "environment": s.Environment, "account": s.Account, "region": s.Region, "operation": operation, "outcome": outcome, "started": start.UTC(), "finished": time.Now().UTC(), "evidence": "command completion; no live inventory asserted"}
	dir := filepath.Join(root, ".deploy", "reports", s.Environment)
	save(filepath.Join(dir, operation+".json"), snapshot)
	md := fmt.Sprintf("# Card Designer %s\n\nProject: %s / %s\n\nOutcome: %s\n\nEvidence: command completion only; resource presence unknown.\n", operation, s.Project, s.Environment, outcome)
	atomic(filepath.Join(dir, operation+".md"), []byte(md))
	atomic(filepath.Join(dir, operation+".html"), []byte("<!doctype html><meta charset=utf-8><title>Deployment report</title><pre>"+md+"</pre>"))
	fmt.Println("Reports:", dir)
}

type buildCache struct{ Input, Artifact string }

func digestFiles(paths []string) (string, error) {
	sort.Strings(paths)
	h := sha256.New()
	for _, p := range paths {
		b, e := os.ReadFile(p)
		if e != nil {
			return "", e
		}
		relative, _ := filepath.Rel(root, p)
		v, _ := json.Marshal(struct {
			Name   string
			Size   int
			Digest string
		}{filepath.ToSlash(relative), len(b), fmt.Sprintf("%x", sha256.Sum256(b))})
		h.Write(v)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func buildFingerprint() (string, error) {
	var paths []string
	for _, dir := range []string{"backend", "web", "packages", "assets", "deploy"} {
		e := filepath.Walk(filepath.Join(root, dir), func(p string, i os.FileInfo, e error) error {
			if e != nil {
				return e
			}
			if i.IsDir() {
				if i.Name() == "node_modules" || i.Name() == "dist" || i.Name() == ".deploy" {
					return filepath.SkipDir
				}
				return nil
			}
			if i.Name() == "bootstrap" || i.Name() == "bootstrap.zip" || strings.Contains(p, "/internal/catalog/catalog.json") {
				return nil
			}
			paths = append(paths, p)
			return nil
		})
		if e != nil {
			return "", e
		}
	}
	digest, e := digestFiles(paths)
	if e != nil {
		return "", e
	}
	gv, e := quiet(root, nil, "go", "version")
	if e != nil {
		return "", e
	}
	nv, e := quiet(root, nil, "node", "--version")
	if e != nil {
		return "", e
	}
	manifest, _ := json.Marshal([]string{"build-v1", digest, string(gv), string(nv), "linux", "arm64", "CGO_ENABLED=0", "-trimpath", "-ldflags=-s -w", "zip-deflate-mode0755"})
	return fmt.Sprintf("%x", sha256.Sum256(manifest)), nil
}
func buildArtifacts() (string, error) {
	paths := []string{filepath.Join(root, "backend", "bootstrap.zip")}
	e := filepath.Walk(filepath.Join(root, "web", "dist"), func(p string, i os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if !i.IsDir() {
			paths = append(paths, p)
		}
		return nil
	})
	if e != nil {
		return "", e
	}
	if len(paths) < 2 {
		return "", errors.New("missing frontend artifacts")
	}
	return digestFiles(paths)
}
func tlsPolicy(raw, bucket string) bool {
	var d struct {
		Statement []struct {
			Effect    string
			Principal any
			Action    any
			Resource  any
			Condition map[string]map[string]any
		}
	}
	if json.Unmarshal([]byte(raw), &d) != nil {
		return false
	}
	contains := func(v any, w string) bool {
		switch a := v.(type) {
		case string:
			return a == w
		case []any:
			for _, x := range a {
				if x == w {
					return true
				}
			}
		}
		return false
	}
	for _, s := range d.Statement {
		principal := s.Principal == "*"
		if m, ok := s.Principal.(map[string]any); ok {
			principal = m["AWS"] == "*"
		}
		condition := s.Condition["Bool"]["aws:SecureTransport"]
		if s.Effect == "Deny" && principal && contains(s.Action, "s3:*") && contains(s.Resource, "arn:aws:s3:::"+bucket) && contains(s.Resource, "arn:aws:s3:::"+bucket+"/*") && (condition == "false" || condition == false) {
			return true
		}
	}
	return false
}

func timeoutAcceptsDefault(timed bool, count int, now, deadline time.Time) bool {
	return timed && count == 0 && !now.Before(deadline)
}

func validateIdentity(project, environment string) error {
	if len(project) > 35 || len(project)+1+len(environment) > 50 {
		return errors.New("project must be at most 35 characters and project-environment at most 50 characters for AWS resource names")
	}
	return nil
}
