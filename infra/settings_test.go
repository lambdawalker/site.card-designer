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
		if err := validateSiteSettings(tc.domain, tc.cert, tc.zone, "123456789012"); (err == nil) != tc.valid {
			t.Fatalf("%+v: %v", tc, err)
		}
	}
	if optionalSetting("none") != "" {
		t.Fatal("generated hostname not supported")
	}
}
