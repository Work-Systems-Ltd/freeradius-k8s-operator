package renderer

import (
	"fmt"
	"regexp"
	"strings"
)

// Validation patterns for FreeRADIUS config values.
// These prevent injection of arbitrary directives into the generated config.
var (
	// safeIdentifier matches module names, NAS types, EAP types, etc.
	safeIdentifierRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,63}$`)

	// safeHostname matches DNS hostnames and IP addresses (no special chars that could break out of quotes).
	safeHostnameRe = regexp.MustCompile(`^[a-zA-Z0-9._:-]{1,253}$`)

	// safeIPOrCIDR matches IPv4/IPv6 addresses with optional CIDR prefix.
	safeIPOrCIDRRe = regexp.MustCompile(`^[0-9a-fA-F.:\/]+$`)

	// safeDN matches LDAP distinguished names (letters, digits, spaces, =, commas, dots).
	safeDNRe = regexp.MustCompile(`^[a-zA-Z0-9 =,._-]+$`)

	// safeFilePath matches file system paths (no shell metacharacters).
	safeFilePathRe = regexp.MustCompile(`^[a-zA-Z0-9/_.-]{1,255}$`)

	// safeURI matches URIs (scheme://host:port/path, no newlines or quotes).
	safeURIRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://[^\s"'\\` + "`" + `]+$`)

	// safeHTTPMethod matches common HTTP methods.
	safeHTTPMethodRe = regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)$`)

	// safeRADIUSAttribute matches RADIUS attribute names (letters, digits, hyphens).
	safeRADIUSAttributeRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]{0,63}$`)

	// safeRADIUSOperator matches valid FreeRADIUS comparison operators.
	safeRADIUSOperatorRe = regexp.MustCompile(`^(==|!=|>=|<=|>|<|=~|!~|\+?=|:=)$`)

	// safeRADIUSValue matches RADIUS attribute values (alphanumeric plus common safe characters).
	safeRADIUSValueRe = regexp.MustCompile(`^[a-zA-Z0-9 _.@:/-]{0,255}$`)

	// safeQuotedString matches values safe to embed in double-quoted FreeRADIUS config strings.
	// Rejects characters that could break out of quotes or inject directives.
	safeQuotedStringRe = regexp.MustCompile(`^[^"'` + "`" + `\\$\n\r]*$`)
)

// ValidateIdentifier checks that s is a safe identifier for use in FreeRADIUS config.
func ValidateIdentifier(field, value string) error {
	if !safeIdentifierRe.MatchString(value) {
		return fmt.Errorf("invalid %s: %q does not match pattern %s", field, value, safeIdentifierRe.String())
	}
	return nil
}

// ValidateHostname checks that s is a safe hostname/IP for use in FreeRADIUS config.
func ValidateHostname(field, value string) error {
	if !safeHostnameRe.MatchString(value) {
		return fmt.Errorf("invalid %s: %q contains disallowed characters", field, value)
	}
	return nil
}

// ValidateIPOrCIDR checks that s is a safe IP or CIDR for use in FreeRADIUS config.
func ValidateIPOrCIDR(field, value string) error {
	if !safeIPOrCIDRRe.MatchString(value) {
		return fmt.Errorf("invalid %s: %q contains disallowed characters", field, value)
	}
	return nil
}

// ValidateDN checks that s is a safe distinguished name.
func ValidateDN(field, value string) error {
	if !safeDNRe.MatchString(value) {
		return fmt.Errorf("invalid %s: %q contains disallowed characters", field, value)
	}
	return nil
}

// ValidateFilePath checks that s is a safe file path.
func ValidateFilePath(field, value string) error {
	if !safeFilePathRe.MatchString(value) {
		return fmt.Errorf("invalid %s: %q contains disallowed characters", field, value)
	}
	return nil
}

// ValidateURI checks that s is a safe URI.
func ValidateURI(field, value string) error {
	if !safeURIRe.MatchString(value) {
		return fmt.Errorf("invalid %s: %q is not a valid URI", field, value)
	}
	return nil
}

// ValidateHTTPMethod checks that s is a valid HTTP method.
func ValidateHTTPMethod(field, value string) error {
	if !safeHTTPMethodRe.MatchString(strings.ToUpper(value)) {
		return fmt.Errorf("invalid %s: %q is not a valid HTTP method", field, value)
	}
	return nil
}

// ValidateQuotedString checks that s is safe to embed in a double-quoted config string.
func ValidateQuotedString(field, value string) error {
	if !safeQuotedStringRe.MatchString(value) {
		return fmt.Errorf("invalid %s: %q contains characters that could break config syntax", field, value)
	}
	return nil
}

// ValidateRADIUSAttribute checks that s is a valid RADIUS attribute name.
func ValidateRADIUSAttribute(field, value string) error {
	if !safeRADIUSAttributeRe.MatchString(value) {
		return fmt.Errorf("invalid %s: %q is not a valid RADIUS attribute name", field, value)
	}
	return nil
}

// ValidateRADIUSOperator checks that s is a valid FreeRADIUS comparison operator.
func ValidateRADIUSOperator(field, value string) error {
	if !safeRADIUSOperatorRe.MatchString(value) {
		return fmt.Errorf("invalid %s: %q is not a valid RADIUS operator", field, value)
	}
	return nil
}

// ValidateRADIUSValue checks that s is a safe RADIUS attribute value.
func ValidateRADIUSValue(field, value string) error {
	if !safeRADIUSValueRe.MatchString(value) {
		return fmt.Errorf("invalid %s: %q contains disallowed characters", field, value)
	}
	return nil
}

// validateClientSpec validates all fields of a ClientSpec before rendering.
func validateClientSpec(c ClientSpec) error {
	if err := ValidateIdentifier("client name", c.Name); err != nil {
		return err
	}
	if err := ValidateIPOrCIDR("client IP", c.IP); err != nil {
		return err
	}
	if c.NASType != "" {
		if err := ValidateIdentifier("nastype", c.NASType); err != nil {
			return err
		}
	}
	return nil
}

// validateSQLConfig validates all fields of a SQLConfig before rendering.
func validateSQLConfig(name string, cfg *SQLConfig) error {
	if err := ValidateIdentifier("SQL dialect", cfg.Dialect); err != nil {
		return err
	}
	if err := ValidateHostname("SQL server", cfg.Server); err != nil {
		return err
	}
	if err := ValidateQuotedString("SQL database", cfg.Database); err != nil {
		return err
	}
	if err := ValidateQuotedString("SQL login", cfg.Login); err != nil {
		return err
	}
	return nil
}

// validateLDAPConfig validates all fields of a LDAPConfig before rendering.
func validateLDAPConfig(name string, cfg *LDAPConfig) error {
	if err := ValidateHostname("LDAP server", cfg.Server); err != nil {
		return err
	}
	if err := ValidateDN("LDAP baseDN", cfg.BaseDN); err != nil {
		return err
	}
	if err := ValidateDN("LDAP identity", cfg.Identity); err != nil {
		return err
	}
	return nil
}

// validateEAPConfig validates all fields of an EAPConfig before rendering.
func validateEAPConfig(name string, cfg *EAPConfig) error {
	if err := ValidateIdentifier("EAP defaultEAPType", cfg.DefaultEAPType); err != nil {
		return err
	}
	if cfg.TLS != nil {
		if cfg.TLS.CertFile != "" {
			if err := ValidateFilePath("EAP TLS certFile", cfg.TLS.CertFile); err != nil {
				return err
			}
		}
		if cfg.TLS.KeyFile != "" {
			if err := ValidateFilePath("EAP TLS keyFile", cfg.TLS.KeyFile); err != nil {
				return err
			}
		}
	}
	if cfg.TTLS != nil {
		if cfg.TTLS.DefaultEAPType != "" {
			if err := ValidateIdentifier("TTLS defaultEAPType", cfg.TTLS.DefaultEAPType); err != nil {
				return err
			}
		}
		if cfg.TTLS.VirtualServer != "" {
			if err := ValidateIdentifier("TTLS virtualServer", cfg.TTLS.VirtualServer); err != nil {
				return err
			}
		}
	}
	if cfg.PEAP != nil {
		if cfg.PEAP.DefaultEAPType != "" {
			if err := ValidateIdentifier("PEAP defaultEAPType", cfg.PEAP.DefaultEAPType); err != nil {
				return err
			}
		}
		if cfg.PEAP.VirtualServer != "" {
			if err := ValidateIdentifier("PEAP virtualServer", cfg.PEAP.VirtualServer); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateRESTConfig validates all fields of a RESTConfig before rendering.
func validateRESTConfig(name string, cfg *RESTConfig) error {
	if err := ValidateURI("REST connectURI", cfg.ConnectURI); err != nil {
		return err
	}
	if cfg.Auth != "" {
		if err := ValidateIdentifier("REST auth", cfg.Auth); err != nil {
			return err
		}
	}
	stages := []*RESTStageConfig{
		cfg.Authorize, cfg.Authenticate, cfg.Preacct,
		cfg.Accounting, cfg.PostAuth, cfg.PreProxy, cfg.PostProxy,
	}
	for _, s := range stages {
		if s == nil {
			continue
		}
		if s.URI != "" {
			if err := ValidateQuotedString("REST stage URI", s.URI); err != nil {
				return err
			}
		}
		if s.Method != "" {
			if err := ValidateHTTPMethod("REST stage method", s.Method); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateRedisConfig validates all fields of a RedisConfig before rendering.
func validateRedisConfig(name string, cfg *RedisConfig) error {
	if err := ValidateHostname("Redis server", cfg.Server); err != nil {
		return err
	}
	return nil
}

// validateModuleConfig validates a module config before rendering.
func validateModuleConfig(mod ModuleConfig) error {
	if err := ValidateIdentifier("module name", mod.Name); err != nil {
		return err
	}
	switch mod.Type {
	case "rlm_sql":
		if mod.SQL != nil {
			return validateSQLConfig(mod.Name, mod.SQL)
		}
	case "rlm_ldap":
		if mod.LDAP != nil {
			return validateLDAPConfig(mod.Name, mod.LDAP)
		}
	case "rlm_eap":
		if mod.EAP != nil {
			return validateEAPConfig(mod.Name, mod.EAP)
		}
	case "rlm_rest":
		if mod.REST != nil {
			return validateRESTConfig(mod.Name, mod.REST)
		}
	case "rlm_redis":
		if mod.Redis != nil {
			return validateRedisConfig(mod.Name, mod.Redis)
		}
	}
	return nil
}

// validatePolicyAction validates a single policy action before rendering.
func validatePolicyAction(a PolicyAction) error {
	switch a.Type {
	case "call":
		if err := ValidateIdentifier("call module", a.Module); err != nil {
			return err
		}
	case "set":
		if err := ValidateRADIUSAttribute("set attribute", a.Attribute); err != nil {
			return err
		}
		if err := ValidateRADIUSValue("set value", a.Value); err != nil {
			return err
		}
	case "reject", "accept":
		// no additional fields to validate
	}
	return nil
}

// validateMatchLeaf validates a single match condition before rendering.
func validateMatchLeaf(leaf MatchLeaf) error {
	if err := ValidateRADIUSAttribute("match attribute", leaf.Attribute); err != nil {
		return err
	}
	if err := ValidateRADIUSOperator("match operator", leaf.Operator); err != nil {
		return err
	}
	if err := ValidateRADIUSValue("match value", leaf.Value); err != nil {
		return err
	}
	return nil
}

// validatePolicySpec validates a full policy spec before rendering.
func validatePolicySpec(p PolicySpec) error {
	if p.Match != nil {
		allLeaves := append(append(p.Match.All, p.Match.Any...), p.Match.None...)
		for _, leaf := range allLeaves {
			if err := validateMatchLeaf(leaf); err != nil {
				return fmt.Errorf("policy %q: %w", p.Name, err)
			}
		}
	}
	for _, a := range p.Actions {
		if err := validatePolicyAction(a); err != nil {
			return fmt.Errorf("policy %q: %w", p.Name, err)
		}
	}
	return nil
}
