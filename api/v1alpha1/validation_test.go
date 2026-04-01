package v1alpha1

import (
	"fmt"
	"regexp"
	"testing"

	"pgregory.net/rapid"
)

// ipCIDRPattern is the same pattern used in the RadiusClientSpec IP field marker.
var ipCIDRPattern = regexp.MustCompile(
	`^((\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?|([0-9a-fA-F:]+)(\/\d{1,3})?)$`,
)

// Feature: freeradius-operator, Property 9: IP and CIDR validation
// Validates: Requirements 4.1
func TestIPCIDRValidation(t *testing.T) {
	// Valid IPv4 addresses must match.
	t.Run("valid IPv4 addresses match", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			octet := rapid.Int32Range(0, 255)
			ip := rapid.Custom(func(t *rapid.T) string {
				return fmt.Sprintf("%d.%d.%d.%d",
					octet.Draw(t, "a"),
					octet.Draw(t, "b"),
					octet.Draw(t, "c"),
					octet.Draw(t, "d"),
				)
			}).Draw(t, "ipv4")
			if !ipCIDRPattern.MatchString(ip) {
				t.Fatalf("expected valid IPv4 %q to match pattern", ip)
			}
		})
	})

	// Valid IPv4 CIDRs must match.
	t.Run("valid IPv4 CIDRs match", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			octet := rapid.Int32Range(0, 255)
			prefix := rapid.Int32Range(0, 32)
			cidr := rapid.Custom(func(t *rapid.T) string {
				return fmt.Sprintf("%d.%d.%d.%d/%d",
					octet.Draw(t, "a"),
					octet.Draw(t, "b"),
					octet.Draw(t, "c"),
					octet.Draw(t, "d"),
					prefix.Draw(t, "prefix"),
				)
			}).Draw(t, "ipv4cidr")
			if !ipCIDRPattern.MatchString(cidr) {
				t.Fatalf("expected valid IPv4 CIDR %q to match pattern", cidr)
			}
		})
	})

	// Valid IPv6 addresses must match.
	t.Run("valid IPv6 addresses match", func(t *testing.T) {
		validIPv6 := []string{
			"2001:db8::1",
			"::1",
			"fe80::1",
			"2001:0db8:0000:0000:0000:0000:0000:0001",
			"::ffff:192.0.2.1",
		}
		for _, addr := range validIPv6 {
			if !ipCIDRPattern.MatchString(addr) {
				t.Fatalf("expected valid IPv6 %q to match pattern", addr)
			}
		}
	})

	// Valid IPv6 CIDRs must match.
	t.Run("valid IPv6 CIDRs match", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			prefix := rapid.Int32Range(0, 128)
			cidr := rapid.Custom(func(t *rapid.T) string {
				return fmt.Sprintf("2001:db8::%d/%d",
					rapid.Int32Range(0, 9999).Draw(t, "suffix"),
					prefix.Draw(t, "prefix"),
				)
			}).Draw(t, "ipv6cidr")
			if !ipCIDRPattern.MatchString(cidr) {
				t.Fatalf("expected valid IPv6 CIDR %q to match pattern", cidr)
			}
		})
	})

	// Strings that are clearly not IP/CIDR must not match.
	t.Run("invalid strings do not match", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate strings that start with a letter — these can never be
			// a valid IPv4 (starts with digit) or IPv6 (hex digits and colons only).
			letter := rapid.RuneRange('g', 'z') // 'g'-'z' are not hex digits
			rest := rapid.StringOfN(rapid.RuneRange('a', 'z'), 1, 20)
			invalid := rapid.Custom(func(t *rapid.T) string {
				return string(letter.Draw(t, "first")) + rest.Draw(t, "rest")
			}).Draw(t, "invalid")
			if ipCIDRPattern.MatchString(invalid) {
				t.Fatalf("expected invalid string %q to not match pattern", invalid)
			}
		})
	})
}

// Feature: freeradius-operator, Property 14: Replicas field validation
// Validates: Requirements 4.1, 8.1
func TestReplicasValidation(t *testing.T) {
	// Any int32 >= 1 must be accepted.
	t.Run("replicas >= 1 are valid", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			r := rapid.Int32Range(1, 1<<30).Draw(t, "replicas")
			if err := validateReplicas(r); err != nil {
				t.Fatalf("expected replicas=%d to be valid, got error: %v", r, err)
			}
		})
	})

	// Zero and negative values must be rejected.
	t.Run("replicas <= 0 are invalid", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			r := rapid.Int32Range(-1<<30, 0).Draw(t, "replicas")
			if err := validateReplicas(r); err == nil {
				t.Fatalf("expected replicas=%d to be invalid, but got no error", r)
			}
		})
	})
}
