package main

import (
	"fmt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"regexp"
	"strings"
)

func optionalSetting(value string) string {
	if strings.EqualFold(value, "none") {
		return ""
	}
	return value
}

// Explicit none clears an old camelCase setting; absent new config preserves it.
func configSetting(c *config.Config, key, legacy string) string {
	value := c.Get(key)
	if value == "" {
		value = c.Get(legacy)
	}
	return optionalSetting(value)
}
func validateSiteSettings(domain, cert, zone, account string) error {
	if domain == "" {
		if cert != "" || zone != "" {
			return fmt.Errorf("certificate and zone require a website hostname")
		}
		return nil
	}
	if len(domain) > 253 || !strings.Contains(domain, ".") {
		return fmt.Errorf("website hostname must be a full DNS hostname")
	}
	label := regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
	for _, part := range strings.Split(domain, ".") {
		if !label.MatchString(part) {
			return fmt.Errorf("invalid website hostname")
		}
	}
	if !regexp.MustCompile(`^arn:aws:acm:us-east-1:` + regexp.QuoteMeta(account) + `:certificate/[A-Za-z0-9-]+$`).MatchString(cert) {
		return fmt.Errorf("custom hostname requires an ISSUED us-east-1 ACM certificate in the deployment account")
	}
	if zone != "" && !regexp.MustCompile(`^Z[A-Z0-9]+$`).MatchString(zone) {
		return fmt.Errorf("invalid Route53 zone ID")
	}
	return nil
}
