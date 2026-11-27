package main

import "testing"

func TestWebsiteSettings(t *testing.T) {
	cert := "arn:aws:acm:us-east-1:123456789012:certificate/abc-123"
	for _, tc := range []struct {
		domain, cert, zone string
		valid              bool
	}{
		{"", "", "", true}, {"cards.example.com", cert, "", true}, {"cards.example.com", cert, "ZABC123", true},
		{"https://cards.example.com", cert, "", false}, {"cards.example.com", "", "", false},
		{"cards.example.com", "arn:aws:acm:us-east-2:123456789012:certificate/abc", "", false},
		{"cards.example.com", "arn:aws:acm:us-east-1:999999999999:certificate/abc", "", false},
		{"", "", "ZABC", false},
	} {
		if err := validateSiteSettings(tc.domain, tc.cert, tc.zone, "123456789012", "route53"); (err == nil) != tc.valid {
			t.Fatalf("%+v: %v", tc, err)
		}
	}
	if optionalSetting("none") != "" {
		t.Fatal("generated hostname not supported")
	}
}

func TestDNSProviderSettings(t *testing.T) {
	cert := "arn:aws:acm:us-east-1:123456789012:certificate/abc-123"
	for _, tc := range []struct {
		provider, zone string
		valid          bool
	}{
		{"", "ZABC123", true}, // Existing Pulumi configurations retain Route 53.
		{"route53", "ZABC123", true},
		{"cloudflare", "023e105f4ecef8ad9ca31a8372d0c353", true},
		{"cloudflare", "", true}, // Manual DNS remains available.
		{"cloudflare", "ZABC123", false},
		{"route53", "023e105f4ecef8ad9ca31a8372d0c353", false},
		{"cloudflare", "not-a-zone", false},
		{"other", "", false},
	} {
		if err := validateSiteSettings("cards.example.com", cert, tc.zone, "123456789012", dnsProvider(tc.provider)); (err == nil) != tc.valid {
			t.Fatalf("%+v: %v", tc, err)
		}
	}
	for _, provider := range []string{"route53", "cloudflare"} {
		if err := validateSiteSettings("", "", "", "123456789012", provider); err != nil {
			t.Fatal("generated URL should not need DNS", err)
		}
	}
}
