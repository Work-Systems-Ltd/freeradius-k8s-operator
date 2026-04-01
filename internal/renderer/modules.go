package renderer

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

func secretFilePath(ref SecretRef) string {
	return fmt.Sprintf("${file:/etc/freeradius/secrets/%s/%s}", ref.Name, ref.Key)
}

func secretFilePathPtr(ref *SecretRef) string {
	if ref == nil {
		return ""
	}
	return secretFilePath(*ref)
}

var tmplFuncs = template.FuncMap{
	"secretFilePath":    secretFilePath,
	"secretFilePathPtr": secretFilePathPtr,
}

var knownModuleTypes = map[string]bool{
	"rlm_sql": true, "rlm_ldap": true, "rlm_eap": true,
	"rlm_rest": true, "rlm_redis": true, "rlm_files": true,
	"rlm_pap": true, "rlm_chap": true, "rlm_mschap": true,
	"rlm_unix": true, "rlm_pam": true, "rlm_python": true,
	"rlm_perl": true, "rlm_cache": true, "rlm_attr_filter": true,
	"rlm_expr": true, "rlm_detail": true, "rlm_linelog": true,
}

var sqlTmpl = template.Must(template.New("sql").Funcs(tmplFuncs).Parse(`sql {{ .Name }} {
    driver = "rlm_sql_{{ .SQL.Dialect }}"
    dialect = "{{ .SQL.Dialect }}"
    server = "{{ .SQL.Server }}"
    port = {{ .SQL.Port }}
    database = "{{ .SQL.Database }}"
    login = "{{ .SQL.Login }}"
    password = "{{ secretFilePath .SQL.PasswordRef }}"
}
`))

var ldapTmpl = template.Must(template.New("ldap").Funcs(tmplFuncs).Parse(`ldap {{ .Name }} {
    server = '{{ .LDAP.Server }}'
    port = {{ .LDAP.Port }}
    base_dn = '{{ .LDAP.BaseDN }}'
    identity = '{{ .LDAP.Identity }}'
    password = {{ secretFilePath .LDAP.PasswordRef }}
}
`))

var eapTmpl = template.Must(template.New("eap").Funcs(tmplFuncs).Parse(`eap {{ .Name }} {
    default_eap_type = {{ .EAP.DefaultEAPType }}
{{ if .EAP.TLS }}    tls {
        certfile = {{ .EAP.TLS.CertFile }}
        keyfile = {{ .EAP.TLS.KeyFile }}
    }
{{ end }}{{ if .EAP.TTLS }}    ttls {
        default_eap_type = {{ .EAP.TTLS.DefaultEAPType }}
{{ if .EAP.TTLS.VirtualServer }}        virtual_server = {{ .EAP.TTLS.VirtualServer }}
{{ end }}    }
{{ end }}{{ if .EAP.PEAP }}    peap {
        default_eap_type = {{ .EAP.PEAP.DefaultEAPType }}
{{ if .EAP.PEAP.VirtualServer }}        virtual_server = {{ .EAP.PEAP.VirtualServer }}
{{ end }}    }
{{ end }}}
`))

var restTmpl = template.Must(template.New("rest").Funcs(tmplFuncs).Parse(`rest {{ .Name }} {
    connect_uri = "{{ .REST.ConnectURI }}"
{{ if .REST.Auth }}    auth = {{ .REST.Auth }}
{{ end }}{{ if .REST.PasswordRef }}    password = "{{ secretFilePathPtr .REST.PasswordRef }}"
{{ end }}{{ if .REST.Authorize }}    authorize {
        uri = "{{ .REST.Authorize.URI }}"
        method = "{{ .REST.Authorize.Method }}"
    }
{{ end }}{{ if .REST.Authenticate }}    authenticate {
        uri = "{{ .REST.Authenticate.URI }}"
        method = "{{ .REST.Authenticate.Method }}"
    }
{{ end }}{{ if .REST.Preacct }}    preacct {
        uri = "{{ .REST.Preacct.URI }}"
        method = "{{ .REST.Preacct.Method }}"
    }
{{ end }}{{ if .REST.Accounting }}    accounting {
        uri = "{{ .REST.Accounting.URI }}"
        method = "{{ .REST.Accounting.Method }}"
    }
{{ end }}{{ if .REST.PostAuth }}    post-auth {
        uri = "{{ .REST.PostAuth.URI }}"
        method = "{{ .REST.PostAuth.Method }}"
    }
{{ end }}{{ if .REST.PreProxy }}    pre-proxy {
        uri = "{{ .REST.PreProxy.URI }}"
        method = "{{ .REST.PreProxy.Method }}"
    }
{{ end }}{{ if .REST.PostProxy }}    post-proxy {
        uri = "{{ .REST.PostProxy.URI }}"
        method = "{{ .REST.PostProxy.Method }}"
    }
{{ end }}}
`))

var redisTmpl = template.Must(template.New("redis").Funcs(tmplFuncs).Parse(`redis {{ .Name }} {
    server = {{ .Redis.Server }}
    port = {{ .Redis.Port }}
{{ if .Redis.PasswordRef }}    password = {{ secretFilePathPtr .Redis.PasswordRef }}
{{ end }}}
`))

var moduleTmpls = map[string]*template.Template{
	"rlm_sql": sqlTmpl, "rlm_ldap": ldapTmpl, "rlm_eap": eapTmpl,
	"rlm_rest": restTmpl, "rlm_redis": redisTmpl,
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
	var buf bytes.Buffer

	if tmpl, ok := moduleTmpls[mod.Type]; ok {
		cfgMissing := map[string]bool{
			"rlm_sql": mod.SQL == nil, "rlm_ldap": mod.LDAP == nil,
			"rlm_eap": mod.EAP == nil, "rlm_rest": mod.REST == nil,
			"rlm_redis": mod.Redis == nil,
		}
		if cfgMissing[mod.Type] {
			return "", fmt.Errorf("%s module %q missing config", mod.Type, mod.Name)
		}
		if err := tmpl.Execute(&buf, mod); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	shortName := strings.TrimPrefix(mod.Type, "rlm_")
	fmt.Fprintf(&buf, "%s %s {\n}\n", shortName, mod.Name)
	return buf.String(), nil
}
