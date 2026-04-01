package renderer

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	safeIdentifierRe      = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,63}$`)
	safeHostnameRe        = regexp.MustCompile(`^[a-zA-Z0-9._:-]{1,253}$`)
	safeIPOrCIDRRe        = regexp.MustCompile(`^[0-9a-fA-F.:\/]+$`)
	safeDNRe              = regexp.MustCompile(`^[a-zA-Z0-9 =,._-]+$`)
	safeFilePathRe        = regexp.MustCompile(`^[a-zA-Z0-9/_.-]{1,255}$`)
	safeURIRe             = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://[^\s"'\\` + "`" + `]+$`)
	safeHTTPMethodRe      = regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)$`)
	safeRADIUSAttributeRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]{0,63}$`)
	safeRADIUSOperatorRe  = regexp.MustCompile(`^(==|!=|>=|<=|>|<|=~|!~|\+?=|:=)$`)
	safeRADIUSValueRe     = regexp.MustCompile(`^[a-zA-Z0-9 _.@:/-]{0,255}$`)
	safeQuotedStringRe    = regexp.MustCompile(`^[^"'` + "`" + `\\$\n\r]*$`)
)

func matchOrErr(re *regexp.Regexp, field, value, hint string) error {
	if !re.MatchString(value) {
		return fmt.Errorf("invalid %s: %q %s", field, value, hint)
	}
	return nil
}

func ValidateIdentifier(field, value string) error {
	return matchOrErr(safeIdentifierRe, field, value, "is not a valid identifier")
}

func ValidateHostname(field, value string) error {
	return matchOrErr(safeHostnameRe, field, value, "contains disallowed characters")
}

func ValidateIPOrCIDR(field, value string) error {
	return matchOrErr(safeIPOrCIDRRe, field, value, "contains disallowed characters")
}

func ValidateDN(field, value string) error {
	return matchOrErr(safeDNRe, field, value, "contains disallowed characters")
}

func ValidateFilePath(field, value string) error {
	return matchOrErr(safeFilePathRe, field, value, "contains disallowed characters")
}

func ValidateURI(field, value string) error {
	return matchOrErr(safeURIRe, field, value, "is not a valid URI")
}

func ValidateHTTPMethod(field, value string) error {
	return matchOrErr(safeHTTPMethodRe, field, strings.ToUpper(value), "is not a valid HTTP method")
}

func ValidateQuotedString(field, value string) error {
	return matchOrErr(safeQuotedStringRe, field, value, "contains characters that could break config syntax")
}

func ValidateRADIUSAttribute(field, value string) error {
	return matchOrErr(safeRADIUSAttributeRe, field, value, "is not a valid RADIUS attribute name")
}

func ValidateRADIUSOperator(field, value string) error {
	return matchOrErr(safeRADIUSOperatorRe, field, value, "is not a valid RADIUS operator")
}

func ValidateRADIUSValue(field, value string) error {
	return matchOrErr(safeRADIUSValueRe, field, value, "contains disallowed characters")
}

func validateClientSpec(c ClientSpec) error {
	if err := ValidateIdentifier("client name", c.Name); err != nil {
		return err
	}
	if err := ValidateIPOrCIDR("client IP", c.IP); err != nil {
		return err
	}
	if c.NASType != "" {
		return ValidateIdentifier("nastype", c.NASType)
	}
	return nil
}

func validateSQLConfig(cfg *SQLConfig) error {
	if err := ValidateIdentifier("SQL dialect", cfg.Dialect); err != nil {
		return err
	}
	if err := ValidateHostname("SQL server", cfg.Server); err != nil {
		return err
	}
	if err := ValidateQuotedString("SQL database", cfg.Database); err != nil {
		return err
	}
	return ValidateQuotedString("SQL login", cfg.Login)
}

func validateLDAPConfig(cfg *LDAPConfig) error {
	if err := ValidateHostname("LDAP server", cfg.Server); err != nil {
		return err
	}
	if err := ValidateDN("LDAP baseDN", cfg.BaseDN); err != nil {
		return err
	}
	return ValidateDN("LDAP identity", cfg.Identity)
}

func validateEAPConfig(cfg *EAPConfig) error {
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
			return ValidateIdentifier("PEAP virtualServer", cfg.PEAP.VirtualServer)
		}
	}
	return nil
}

func validateRESTConfig(cfg *RESTConfig) error {
	if err := ValidateURI("REST connectURI", cfg.ConnectURI); err != nil {
		return err
	}
	if cfg.Auth != "" {
		if err := ValidateIdentifier("REST auth", cfg.Auth); err != nil {
			return err
		}
	}
	for _, s := range []*RESTStageConfig{
		cfg.Authorize, cfg.Authenticate, cfg.Preacct,
		cfg.Accounting, cfg.PostAuth, cfg.PreProxy, cfg.PostProxy,
	} {
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

func validateRedisConfig(cfg *RedisConfig) error {
	return ValidateHostname("Redis server", cfg.Server)
}

func validateModuleConfig(mod ModuleConfig) error {
	if err := ValidateIdentifier("module name", mod.Name); err != nil {
		return err
	}
	switch mod.Type {
	case "rlm_sql":
		if mod.SQL != nil {
			return validateSQLConfig(mod.SQL)
		}
	case "rlm_ldap":
		if mod.LDAP != nil {
			return validateLDAPConfig(mod.LDAP)
		}
	case "rlm_eap":
		if mod.EAP != nil {
			return validateEAPConfig(mod.EAP)
		}
	case "rlm_rest":
		if mod.REST != nil {
			return validateRESTConfig(mod.REST)
		}
	case "rlm_redis":
		if mod.Redis != nil {
			return validateRedisConfig(mod.Redis)
		}
	}
	return nil
}

func validatePolicyAction(a PolicyAction) error {
	switch a.Type {
	case "call":
		return ValidateIdentifier("call module", a.Module)
	case "set":
		if err := ValidateRADIUSAttribute("set attribute", a.Attribute); err != nil {
			return err
		}
		return ValidateRADIUSValue("set value", a.Value)
	}
	return nil
}

func validateMatchLeaf(leaf MatchLeaf) error {
	if err := ValidateRADIUSAttribute("match attribute", leaf.Attribute); err != nil {
		return err
	}
	if err := ValidateRADIUSOperator("match operator", leaf.Operator); err != nil {
		return err
	}
	return ValidateRADIUSValue("match value", leaf.Value)
}

func validatePolicySpec(p PolicySpec) error {
	if p.Match != nil {
		for _, leaf := range append(append(p.Match.All, p.Match.Any...), p.Match.None...) {
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
