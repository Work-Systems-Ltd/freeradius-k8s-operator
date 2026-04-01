package renderer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGolden_SQLModuleRendering(t *testing.T) {
	files, err := New().Render(RenderContext{
		Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3", Modules: []ModuleConfig{{
			Name: "sql", Type: "rlm_sql", Enabled: true,
			SQL: &SQLConfig{Dialect: "postgresql", Server: "postgres.example.com", Port: 5432,
				Database: "radius", Login: "radius", PasswordRef: SecretRef{Name: "radius-sql-secret", Key: "password"}},
		}}},
	})
	require.NoError(t, err)
	c := files["mods-enabled/sql"]
	assert.Contains(t, c, `driver = "rlm_sql_postgresql"`)
	assert.Contains(t, c, `dialect = "postgresql"`)
	assert.Contains(t, c, `server = "postgres.example.com"`)
	assert.Contains(t, c, `port = 5432`)
	assert.Contains(t, c, `database = "radius"`)
	assert.Contains(t, c, `login = "radius"`)
	assert.Contains(t, c, `${file:/etc/freeradius/secrets/radius-sql-secret/password}`)
}

func TestGolden_LDAPModuleRendering(t *testing.T) {
	files, err := New().Render(RenderContext{
		Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3", Modules: []ModuleConfig{{
			Name: "ldap", Type: "rlm_ldap", Enabled: true,
			LDAP: &LDAPConfig{Server: "ldap.corp.example.com", Port: 389, BaseDN: "dc=corp,dc=example,dc=com",
				Identity: "cn=radius,dc=corp,dc=example,dc=com", PasswordRef: SecretRef{Name: "radius-ldap-secret", Key: "password"}},
		}}},
	})
	require.NoError(t, err)
	c := files["mods-enabled/ldap"]
	assert.Contains(t, c, "base_dn")
	assert.Contains(t, c, "dc=corp,dc=example,dc=com")
	assert.Contains(t, c, "${file:/etc/freeradius/secrets/radius-ldap-secret/password}")
}

func TestGolden_ClientsConfContainsAllClients(t *testing.T) {
	clients := []ClientSpec{
		{Name: "bng-auckland-01", IP: "10.0.1.0/24", SecretRef: SecretRef{Name: "bng-auckland-01-secret", Key: "shared-secret"}, NASType: "nokia"},
		{Name: "bng-sydney-01", IP: "10.0.2.0/24", SecretRef: SecretRef{Name: "bng-sydney-01-secret", Key: "shared-secret"}, NASType: "cisco"},
		{Name: "bng-melbourne-01", IP: "10.0.3.1", SecretRef: SecretRef{Name: "bng-melbourne-01-secret", Key: "shared-secret"}, NASType: "other"},
	}
	files, err := New().Render(RenderContext{Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"}, Clients: clients})
	require.NoError(t, err)
	c := files["clients.conf"]
	for _, cl := range clients {
		assert.Contains(t, c, "client "+cl.Name+" {")
		assert.Contains(t, c, "ipaddr = "+cl.IP)
		assert.Contains(t, c, "${file:/etc/freeradius/secrets/"+cl.SecretRef.Name+"/"+cl.SecretRef.Key+"}")
		assert.Contains(t, c, "nastype = "+cl.NASType)
	}
}

func TestGolden_IPv6ClientUsesIpv6addr(t *testing.T) {
	files, err := New().Render(RenderContext{
		Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"},
		Clients: []ClientSpec{
			{Name: "v4-client", IP: "10.0.1.1", SecretRef: SecretRef{Name: "s1", Key: "k"}, NASType: "other"},
			{Name: "v6-client", IP: "2001:db8::1", SecretRef: SecretRef{Name: "s2", Key: "k"}, NASType: "other"},
			{Name: "v6-mapped", IP: "::ffff:192.0.2.1", SecretRef: SecretRef{Name: "s3", Key: "k"}, NASType: "other"},
		},
	})
	require.NoError(t, err)
	c := files["clients.conf"]
	assert.Contains(t, c, "client v4-client {\n    ipaddr = 10.0.1.1")
	assert.Contains(t, c, "client v6-client {\n    ipv6addr = 2001:db8::1")
	assert.Contains(t, c, "client v6-mapped {\n    ipv6addr = ::ffff:192.0.2.1")
}

func TestGolden_SitesEnabledContainsAllStages(t *testing.T) {
	files, err := New().Render(RenderContext{Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"}})
	require.NoError(t, err)
	s := files["sites-enabled/default"]
	for _, stage := range validStages {
		assert.Contains(t, s, "    "+stage+" {")
	}
	assert.Contains(t, s, "server default {")
}

func TestGolden_DisabledModulesProduceNoOutput(t *testing.T) {
	files, err := New().Render(RenderContext{Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3", Modules: []ModuleConfig{
		{Name: "sql", Type: "rlm_sql", Enabled: false, SQL: &SQLConfig{Dialect: "postgresql", Server: "db", Port: 5432, Database: "r", Login: "r", PasswordRef: SecretRef{Name: "s", Key: "p"}}},
		{Name: "ldap", Type: "rlm_ldap", Enabled: false, LDAP: &LDAPConfig{Server: "ldap", Port: 389, BaseDN: "dc=e", Identity: "cn=a", PasswordRef: SecretRef{Name: "s", Key: "p"}}},
	}}})
	require.NoError(t, err)
	_, hasSql := files["mods-enabled/sql"]
	_, hasLdap := files["mods-enabled/ldap"]
	assert.False(t, hasSql)
	assert.False(t, hasLdap)
}

func TestGolden_EmptyClientListProducesLocalhostOnly(t *testing.T) {
	files, err := New().Render(RenderContext{Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"}})
	require.NoError(t, err)
	c := files["clients.conf"]
	assert.Contains(t, c, "client localhost {")
	assert.Contains(t, c, "ipaddr = 127.0.0.1")
	assert.Contains(t, c, "client localhost_v6 {")
	assert.Contains(t, c, "ipv6addr = ::1")
	assert.Equal(t, 2, strings.Count(c, "client "))
}

func TestGolden_RadiusdConfHasRequiredDirectives(t *testing.T) {
	files, err := New().Render(RenderContext{Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"}})
	require.NoError(t, err)
	r := files["radiusd.conf"]
	assert.Contains(t, r, "status_server = yes")
	assert.Contains(t, r, "$INCLUDE clients.conf")
	assert.Contains(t, r, "$INCLUDE mods-enabled/")
	assert.Contains(t, r, "$INCLUDE sites-enabled/")
	assert.Contains(t, r, "ipaddr = *")
	assert.Contains(t, r, "ipv6addr = ::")
	assert.Contains(t, r, "port = 1812")
	assert.Contains(t, r, "port = 1813")
}

func TestGolden_InvalidModuleTypeReturnsError(t *testing.T) {
	_, err := New().Render(RenderContext{Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3", Modules: []ModuleConfig{
		{Name: "unknown", Type: "rlm_nonexistent", Enabled: true},
	}}})
	require.Error(t, err)
	var modErr *InvalidModuleError
	require.ErrorAs(t, err, &modErr)
	assert.Equal(t, "rlm_nonexistent", modErr.ModuleType)
}

func TestGolden_PoliciesRenderedInPriorityOrder(t *testing.T) {
	files, err := New().Render(RenderContext{
		Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"},
		Policies: []PolicySpec{
			{Name: "policy-high", Stage: "authorize", Priority: 100, Actions: []PolicyAction{{Type: "accept"}}},
			{Name: "policy-low", Stage: "authorize", Priority: 10, Actions: []PolicyAction{{Type: "accept"}}},
			{Name: "policy-mid", Stage: "authorize", Priority: 50, Actions: []PolicyAction{{Type: "accept"}}},
		},
	})
	require.NoError(t, err)
	s := files["sites-enabled/default"]
	assert.Less(t, strings.Index(s, "policy-low"), strings.Index(s, "policy-mid"))
	assert.Less(t, strings.Index(s, "policy-mid"), strings.Index(s, "policy-high"))
}

func TestGolden_EAPModuleRendering(t *testing.T) {
	files, err := New().Render(RenderContext{Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3", Modules: []ModuleConfig{{
		Name: "eap", Type: "rlm_eap", Enabled: true,
		EAP: &EAPConfig{DefaultEAPType: "peap",
			PEAP: &EAPPEAPConfig{DefaultEAPType: "mschapv2"},
			TLS:  &EAPTLSConfig{CertFile: "/etc/freeradius/certs/server.pem", KeyFile: "/etc/freeradius/certs/server.key"},
		},
	}}}})
	require.NoError(t, err)
	c := files["mods-enabled/eap"]
	assert.Contains(t, c, "default_eap_type = peap")
	assert.Contains(t, c, "peap {")
	assert.Contains(t, c, "tls {")
}
