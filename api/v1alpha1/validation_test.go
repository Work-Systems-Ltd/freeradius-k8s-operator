package v1alpha1

import (
	"fmt"
	"regexp"
	"testing"

	"pgregory.net/rapid"
)

var ipCIDRPattern = regexp.MustCompile(
	`^((\d{1,3}\.){3}\d{1,3}(/\d{1,2})?|[0-9a-fA-F:.]*:[0-9a-fA-F:.]*(/\d{1,3})?)$`,
)

func TestIPCIDRValidation(t *testing.T) {
	t.Run("valid IPv4 addresses match", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			octet := rapid.Int32Range(0, 255)
			ip := fmt.Sprintf("%d.%d.%d.%d",
				octet.Draw(t, "a"), octet.Draw(t, "b"),
				octet.Draw(t, "c"), octet.Draw(t, "d"))
			if !ipCIDRPattern.MatchString(ip) {
				t.Fatalf("expected valid IPv4 %q to match", ip)
			}
		})
	})

	t.Run("valid IPv4 CIDRs match", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			octet := rapid.Int32Range(0, 255)
			cidr := fmt.Sprintf("%d.%d.%d.%d/%d",
				octet.Draw(t, "a"), octet.Draw(t, "b"),
				octet.Draw(t, "c"), octet.Draw(t, "d"),
				rapid.Int32Range(0, 32).Draw(t, "prefix"))
			if !ipCIDRPattern.MatchString(cidr) {
				t.Fatalf("expected valid IPv4 CIDR %q to match", cidr)
			}
		})
	})

	t.Run("valid IPv6 addresses match", func(t *testing.T) {
		for _, addr := range []string{"2001:db8::1", "::1", "fe80::1", "2001:0db8:0000:0000:0000:0000:0000:0001", "::ffff:192.0.2.1"} {
			if !ipCIDRPattern.MatchString(addr) {
				t.Fatalf("expected valid IPv6 %q to match", addr)
			}
		}
	})

	t.Run("valid IPv6 CIDRs match", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cidr := fmt.Sprintf("2001:db8::%d/%d",
				rapid.Int32Range(0, 9999).Draw(t, "suffix"),
				rapid.Int32Range(0, 128).Draw(t, "prefix"))
			if !ipCIDRPattern.MatchString(cidr) {
				t.Fatalf("expected valid IPv6 CIDR %q to match", cidr)
			}
		})
	})

	t.Run("invalid strings do not match", func(t *testing.T) {
		for _, s := range []string{"hello", "not-an-ip", "xyz.abc", "g1234", "just spaces", "abcdef", "a", "FF"} {
			if ipCIDRPattern.MatchString(s) {
				t.Fatalf("expected %q to not match", s)
			}
		}
	})
}

func TestReplicasValidation(t *testing.T) {
	t.Run("replicas >= 1 are valid", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			r := rapid.Int32Range(1, 1<<30).Draw(t, "replicas")
			if err := validateReplicas(r); err != nil {
				t.Fatalf("expected replicas=%d to be valid, got: %v", r, err)
			}
		})
	})

	t.Run("replicas <= 0 are invalid", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			r := rapid.Int32Range(-1<<30, 0).Draw(t, "replicas")
			if err := validateReplicas(r); err == nil {
				t.Fatalf("expected replicas=%d to be invalid", r)
			}
		})
	})
}
