package renderer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGolden_SQLModuleRendering(t *testing.T) {
	r := New()
	ctx := RenderContext{
		Cluster: ClusterSpec{
			Replicas: 1,
			Image:    "docker.io/freeradius/freeradius-server:3.2.3",
			Modules: []ModuleConfig{
				{
					Name:    "sql",
					Type:    "rlm_sql",
					Enabled: true,
					SQL: &SQLConfig{
						Dialect:  "postgresql",
						Server:   "postgres.example.com",
						Port:     5432,
						Database: "radius",
						Login:    "radius",
						PasswordRef: SecretRef{
							Name: "radius-sql-secret",
							Key:  "password",
						},
					},
				},
			},
		},
	}

	files, err := r.Render(ctx)
	require.NoError(t, err)

	sqlContent := files["mods-enabled/sql"]
	require.NotEmpty(t, sqlContent, "mods-enabled/sql should not be empty")

	assert.Contains(t, sqlContent, `driver = "rlm_sql_postgresql"`)
	assert.Contains(t, sqlContent, `dialect = "postgresql"`)
	assert.Contains(t, sqlContent, `server = "postgres.example.com"`)
	assert.Contains(t, sqlContent, `port = 5432`)
	assert.Contains(t, sqlContent, `database = "radius"`)
	assert.Contains(t, sqlContent, `login = "radius"`)
	assert.Contains(t, sqlContent, `${file:/etc/freeradius/secrets/radius-sql-secret/password}`)
	// Must NOT contain plaintext password
	assert.NotContains(t, sqlContent, "supersecret")
}

func TestGolden_LDAPModuleRendering(t *testing.T) {
	r := New()
	ctx := RenderContext{
		Cluster: ClusterSpec{
			Replicas: 1,
			Image:    "docker.io/freeradius/freeradius-server:3.2.3",
			Modules: []ModuleConfig{
				{
					Name:    "ldap",
					Type:    "rlm_ldap",
					Enabled: true,
					LDAP: &LDAPConfig{
						Server:   "ldap.corp.example.com",
						Port:     389,
						BaseDN:   "dc=corp,dc=example,dc=com",
						Identity: "cn=radius,dc=corp,dc=example,dc=com",
						PasswordRef: SecretRef{
							Name: "radius-ldap-secret",
							Key:  "password",
						},
					},
				},
			},
		},
	}

	files, err := r.Render(ctx)
	require.NoError(t, err)

	ldapContent := files["mods-enabled/ldap"]
	require.NotEmpty(t, ldapContent, "mods-enabled/ldap should not be empty")

	assert.Contains(t, ldapContent, "base_dn")
	assert.Contains(t, ldapContent, "dc=corp,dc=example,dc=com")
	assert.Contains(t, ldapContent, "identity")
	assert.Contains(t, ldapContent, "${file:/etc/freeradius/secrets/radius-ldap-secret/password}")
}

func TestGolden_ClientsConfContainsAllClients(t *testing.T) {
	r := New()
	clients := []ClientSpec{
		{Name: "bng-auckland-01", IP: "10.0.1.0/24", SecretRef: SecretRef{Name: "bng-auckland-01-secret", Key: "shared-secret"}, NASType: "nokia"},
		{Name: "bng-sydney-01", IP: "10.0.2.0/24", SecretRef: SecretRef{Name: "bng-sydney-01-secret", Key: "shared-secret"}, NASType: "cisco"},
		{Name: "bng-melbourne-01", IP: "10.0.3.1", SecretRef: SecretRef{Name: "bng-melbourne-01-secret", Key: "shared-secret"}, NASType: "other"},
	}

	ctx := RenderContext{
		Cluster:  ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"},
		Clients:  clients,
		Policies: nil,
	}

	files, err := r.Render(ctx)
	require.NoError(t, err)

	clientsConf := files["clients.conf"]
	require.NotEmpty(t, clientsConf)

	for _, c := range clients {
		assert.Contains(t, clientsConf, "client "+c.Name+" {")
		assert.Contains(t, clientsConf, "ipaddr = "+c.IP)
		assert.Contains(t, clientsConf, "${file:/etc/freeradius/secrets/"+c.SecretRef.Name+"/"+c.SecretRef.Key+"}")
		assert.Contains(t, clientsConf, "nastype = "+c.NASType)
	}
}

func TestGolden_SitesEnabledContainsAllStages(t *testing.T) {
	r := New()
	ctx := RenderContext{
		Cluster:  ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"},
		Policies: nil,
	}

	files, err := r.Render(ctx)
	require.NoError(t, err)

	sitesDefault := files["sites-enabled/default"]
	require.NotEmpty(t, sitesDefault)

	for _, stage := range validStages {
		assert.Contains(t, sitesDefault, "    "+stage+" {", "stage %q missing", stage)
	}
	assert.Contains(t, sitesDefault, "server default {")
}

func TestGolden_DisabledModulesProduceNoOutput(t *testing.T) {
	r := New()
	ctx := RenderContext{
		Cluster: ClusterSpec{
			Replicas: 1,
			Image:    "freeradius:3.2.3",
			Modules: []ModuleConfig{
				{
					Name:    "sql",
					Type:    "rlm_sql",
					Enabled: false,
					SQL: &SQLConfig{
						Dialect:     "postgresql",
						Server:      "db.example.com",
						Port:        5432,
						Database:    "radius",
						Login:       "radius",
						PasswordRef: SecretRef{Name: "secret", Key: "password"},
					},
				},
				{
					Name:    "ldap",
					Type:    "rlm_ldap",
					Enabled: false,
					LDAP: &LDAPConfig{
						Server:      "ldap.example.com",
						Port:        389,
						BaseDN:      "dc=example,dc=com",
						Identity:    "cn=admin,dc=example,dc=com",
						PasswordRef: SecretRef{Name: "secret", Key: "password"},
					},
				},
			},
		},
	}

	files, err := r.Render(ctx)
	require.NoError(t, err)

	_, hasSql := files["mods-enabled/sql"]
	assert.False(t, hasSql, "disabled sql module should not appear in output")

	_, hasLdap := files["mods-enabled/ldap"]
	assert.False(t, hasLdap, "disabled ldap module should not appear in output")
}

func TestGolden_EmptyClientListProducesLocalhostOnly(t *testing.T) {
	r := New()
	ctx := RenderContext{
		Cluster:  ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"},
		Clients:  nil,
		Policies: nil,
	}

	files, err := r.Render(ctx)
	require.NoError(t, err)

	clientsConf := files["clients.conf"]
	require.NotEmpty(t, clientsConf, "clients.conf should have at least a header")

	// Should contain the localhost client for readiness probes
	assert.Contains(t, clientsConf, "client localhost {")
	assert.Contains(t, clientsConf, "secret = testing123")
	// Should contain the header comment
	assert.Contains(t, clientsConf, "clients.conf")
	// Should have exactly one client block (localhost)
	assert.Equal(t, 1, strings.Count(clientsConf, "client "))
}

func TestGolden_RadiusdConfHasRequiredDirectives(t *testing.T) {
	r := New()
	ctx := RenderContext{
		Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"},
	}

	files, err := r.Render(ctx)
	require.NoError(t, err)

	radiusd := files["radiusd.conf"]
	assert.Contains(t, radiusd, "status_server = yes")
	assert.Contains(t, radiusd, "$INCLUDE clients.conf")
	assert.Contains(t, radiusd, "$INCLUDE mods-enabled/")
	assert.Contains(t, radiusd, "$INCLUDE sites-enabled/")
	assert.Contains(t, radiusd, "port = 1812")
	assert.Contains(t, radiusd, "port = 1813")
}

func TestGolden_InvalidModuleTypeReturnsError(t *testing.T) {
	r := New()
	ctx := RenderContext{
		Cluster: ClusterSpec{
			Replicas: 1,
			Image:    "freeradius:3.2.3",
			Modules: []ModuleConfig{
				{
					Name:    "unknown",
					Type:    "rlm_nonexistent",
					Enabled: true,
				},
			},
		},
	}

	_, err := r.Render(ctx)
	require.Error(t, err)

	var modErr *InvalidModuleError
	require.ErrorAs(t, err, &modErr)
	assert.Equal(t, "rlm_nonexistent", modErr.ModuleType)
}

func TestGolden_PoliciesRenderedInPriorityOrder(t *testing.T) {
	r := New()
	ctx := RenderContext{
		Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"},
		Policies: []PolicySpec{
			{Name: "policy-high", Stage: "authorize", Priority: 100, Actions: []PolicyAction{{Type: "accept"}}},
			{Name: "policy-low", Stage: "authorize", Priority: 10, Actions: []PolicyAction{{Type: "accept"}}},
			{Name: "policy-mid", Stage: "authorize", Priority: 50, Actions: []PolicyAction{{Type: "accept"}}},
		},
	}

	files, err := r.Render(ctx)
	require.NoError(t, err)

	sitesDefault := files["sites-enabled/default"]

	posLow := strings.Index(sitesDefault, "policy-low")
	posMid := strings.Index(sitesDefault, "policy-mid")
	posHigh := strings.Index(sitesDefault, "policy-high")

	assert.Less(t, posLow, posMid, "policy-low (priority 10) should appear before policy-mid (priority 50)")
	assert.Less(t, posMid, posHigh, "policy-mid (priority 50) should appear before policy-high (priority 100)")
}

func TestGolden_EAPModuleRendering(t *testing.T) {
	r := New()
	ctx := RenderContext{
		Cluster: ClusterSpec{
			Replicas: 1,
			Image:    "freeradius:3.2.3",
			Modules: []ModuleConfig{
				{
					Name:    "eap",
					Type:    "rlm_eap",
					Enabled: true,
					EAP: &EAPConfig{
						DefaultEAPType: "peap",
						PEAP: &EAPPEAPConfig{
							DefaultEAPType: "mschapv2",
						},
						TLS: &EAPTLSConfig{
							CertFile: "/etc/freeradius/certs/server.pem",
							KeyFile:  "/etc/freeradius/certs/server.key",
						},
					},
				},
			},
		},
	}

	files, err := r.Render(ctx)
	require.NoError(t, err)

	eapContent := files["mods-enabled/eap"]
	require.NotEmpty(t, eapContent)

	assert.Contains(t, eapContent, "default_eap_type = peap")
	assert.Contains(t, eapContent, "peap {")
	assert.Contains(t, eapContent, "tls {")
}
