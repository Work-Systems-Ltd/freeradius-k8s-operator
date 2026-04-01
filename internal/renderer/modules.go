package renderer

import (
	"bytes"
	"fmt"
	"strings"
)

var knownModuleTypes = map[string]bool{
	"rlm_sql": true, "rlm_ldap": true, "rlm_eap": true,
	"rlm_rest": true, "rlm_redis": true, "rlm_files": true,
	"rlm_pap": true, "rlm_chap": true, "rlm_mschap": true,
	"rlm_unix": true, "rlm_pam": true, "rlm_python": true,
	"rlm_perl": true, "rlm_cache": true, "rlm_attr_filter": true,
	"rlm_expr": true, "rlm_detail": true, "rlm_linelog": true,
}

var moduleTemplateNames = map[string]string{
	"rlm_sql": "sql.tmpl", "rlm_ldap": "ldap.tmpl", "rlm_eap": "eap.tmpl",
	"rlm_rest": "rest.tmpl", "rlm_redis": "redis.tmpl",
}

func renderModules(modules []ModuleConfig) (ConfigFiles, error) {
	files := make(ConfigFiles)
	for _, mod := range modules {
		if !mod.Enabled {
			continue
		}
		if !knownModuleTypes[mod.Type] {
			return nil, &InvalidModuleError{ModuleType: mod.Type}
		}
		content, err := renderModule(mod)
		if err != nil {
			return nil, fmt.Errorf("rendering module %q (type %s): %w", mod.Name, mod.Type, err)
		}
		files["mods-enabled/"+mod.Name] = content
	}
	return files, nil
}

func renderModule(mod ModuleConfig) (string, error) {
	tmplName, hasTemplate := moduleTemplateNames[mod.Type]
	if !hasTemplate {
		shortName := strings.TrimPrefix(mod.Type, "rlm_")
		return fmt.Sprintf("%s %s {\n}\n", shortName, mod.Name), nil
	}

	cfgMissing := map[string]bool{
		"rlm_sql": mod.SQL == nil, "rlm_ldap": mod.LDAP == nil,
		"rlm_eap": mod.EAP == nil, "rlm_rest": mod.REST == nil,
		"rlm_redis": mod.Redis == nil,
	}
	if cfgMissing[mod.Type] {
		return "", fmt.Errorf("%s module %q missing config", mod.Type, mod.Name)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, tmplName, mod); err != nil {
		return "", err
	}
	return buf.String(), nil
}
