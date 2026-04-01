package renderer

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func genSecretRef(t *rapid.T) SecretRef {
	return SecretRef{
		Name: rapid.StringMatching(`[a-z][a-z0-9-]{0,20}`).Draw(t, "secretName"),
		Key:  rapid.StringMatching(`[a-z][a-z0-9-]{0,20}`).Draw(t, "secretKey"),
	}
}

func genClientSpec(t *rapid.T) ClientSpec {
	return ClientSpec{
		Name:      rapid.StringMatching(`[a-z][a-z0-9-]{0,20}`).Draw(t, "clientName"),
		IP:        rapid.StringMatching(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`).Draw(t, "clientIP"),
		SecretRef: genSecretRef(t),
		NASType:   rapid.SampledFrom([]string{"cisco", "nokia", "other", "juniper"}).Draw(t, "nasType"),
	}
}

func genMatchLeaf(t *rapid.T) MatchLeaf {
	return MatchLeaf{
		Attribute: rapid.SampledFrom([]string{"Auth-Type", "User-Name", "NAS-IP-Address", "Service-Type"}).Draw(t, "attr"),
		Operator:  rapid.SampledFrom([]string{"==", "!=", ">=", "<="}).Draw(t, "op"),
		Value:     rapid.StringMatching(`[A-Za-z0-9_-]{1,20}`).Draw(t, "val"),
	}
}

func genPolicyMatch(t *rapid.T) *PolicyMatch {
	if !rapid.Bool().Draw(t, "hasMatch") {
		return nil
	}
	kind := rapid.IntRange(0, 2).Draw(t, "matchKind")
	n := rapid.IntRange(1, 3).Draw(t, "leafCount")
	leaves := make([]MatchLeaf, n)
	for i := range leaves {
		leaves[i] = genMatchLeaf(t)
	}
	switch kind {
	case 0:
		return &PolicyMatch{All: leaves}
	case 1:
		return &PolicyMatch{Any: leaves}
	default:
		return &PolicyMatch{None: leaves}
	}
}

func genPolicyAction(t *rapid.T) PolicyAction {
	actionType := rapid.SampledFrom([]string{"set", "call", "reject", "accept"}).Draw(t, "actionType")
	switch actionType {
	case "set":
		return PolicyAction{
			Type:      "set",
			Attribute: rapid.SampledFrom([]string{"Reply-Message", "Session-Timeout", "Framed-IP-Address"}).Draw(t, "setAttr"),
			Value:     rapid.StringMatching(`[A-Za-z0-9 ]{1,20}`).Draw(t, "setValue"),
		}
	case "call":
		return PolicyAction{
			Type:   "call",
			Module: rapid.SampledFrom([]string{"pap", "chap", "mschap", "ldap", "sql"}).Draw(t, "module"),
		}
	default:
		return PolicyAction{Type: actionType}
	}
}

func genPolicySpec(t *rapid.T, stage string) PolicySpec {
	n := rapid.IntRange(0, 3).Draw(t, "actionCount")
	actions := make([]PolicyAction, n)
	for i := range actions {
		actions[i] = genPolicyAction(t)
	}
	return PolicySpec{
		Name:     rapid.StringMatching(`[a-z][a-z0-9-]{0,20}`).Draw(t, "policyName"),
		Stage:    stage,
		Priority: rapid.Int32Range(0, 1000).Draw(t, "priority"),
		Match:    genPolicyMatch(t),
		Actions:  actions,
	}
}

func genSQLConfig(t *rapid.T) *SQLConfig {
	return &SQLConfig{
		Dialect:     rapid.SampledFrom([]string{"mysql", "postgresql", "sqlite", "mssql"}).Draw(t, "dialect"),
		Server:      rapid.StringMatching(`[a-z][a-z0-9.-]{0,30}`).Draw(t, "sqlServer"),
		Port:        rapid.Int32Range(1024, 65535).Draw(t, "sqlPort"),
		Database:    rapid.StringMatching(`[a-z][a-z0-9_]{0,20}`).Draw(t, "sqlDB"),
		Login:       rapid.StringMatching(`[a-z][a-z0-9_]{0,20}`).Draw(t, "sqlLogin"),
		PasswordRef: genSecretRef(t),
	}
}

func genLDAPConfig(t *rapid.T) *LDAPConfig {
	return &LDAPConfig{
		Server: rapid.StringMatching(`[a-z][a-z0-9.-]{0,30}`).Draw(t, "ldapServer"),
		Port:   rapid.Int32Range(389, 636).Draw(t, "ldapPort"),
		BaseDN: "dc=example,dc=com", Identity: "cn=admin,dc=example,dc=com",
		PasswordRef: genSecretRef(t),
	}
}

func genEAPConfig(t *rapid.T) *EAPConfig {
	return &EAPConfig{DefaultEAPType: rapid.SampledFrom([]string{"peap", "ttls", "tls", "md5"}).Draw(t, "eapType")}
}

func genRedisConfig(t *rapid.T) *RedisConfig {
	cfg := &RedisConfig{
		Server: rapid.StringMatching(`[a-z][a-z0-9.-]{0,30}`).Draw(t, "redisServer"),
		Port:   rapid.Int32Range(1024, 65535).Draw(t, "redisPort"),
	}
	if rapid.Bool().Draw(t, "redisHasPassword") {
		ref := genSecretRef(t)
		cfg.PasswordRef = &ref
	}
	return cfg
}

func genModuleConfig(t *rapid.T) ModuleConfig {
	modType := rapid.SampledFrom([]string{
		"rlm_sql", "rlm_ldap", "rlm_eap", "rlm_redis",
		"rlm_files", "rlm_pap", "rlm_chap", "rlm_mschap",
		"rlm_unix", "rlm_pam", "rlm_cache", "rlm_expr",
	}).Draw(t, "modType")

	mod := ModuleConfig{
		Name: rapid.StringMatching(`[a-z][a-z0-9_]{0,15}`).Draw(t, "modName"),
		Type: modType, Enabled: rapid.Bool().Draw(t, "modEnabled"),
	}
	switch modType {
	case "rlm_sql":
		mod.SQL = genSQLConfig(t)
	case "rlm_ldap":
		mod.LDAP = genLDAPConfig(t)
	case "rlm_eap":
		mod.EAP = genEAPConfig(t)
	case "rlm_redis":
		mod.Redis = genRedisConfig(t)
	}
	return mod
}

func genRenderContext(t *rapid.T) RenderContext {
	nClients := rapid.IntRange(0, 5).Draw(t, "nClients")
	clients := make([]ClientSpec, nClients)
	for i := range clients {
		clients[i] = genClientSpec(t)
	}
	nMods := rapid.IntRange(0, 4).Draw(t, "nMods")
	modules := make([]ModuleConfig, nMods)
	for i := range modules {
		modules[i] = genModuleConfig(t)
	}
	nPolicies := rapid.IntRange(0, 5).Draw(t, "nPolicies")
	policies := make([]PolicySpec, nPolicies)
	for i := range policies {
		policies[i] = genPolicySpec(t, rapid.SampledFrom(validStages).Draw(t, "stage"))
	}
	return RenderContext{
		Cluster:  ClusterSpec{Replicas: rapid.Int32Range(1, 10).Draw(t, "replicas"), Image: "freeradius:3.2.3", Modules: modules},
		Clients:  clients,
		Policies: policies,
	}
}

func TestRendererOutputFiles(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		ctx := genRenderContext(t)
		files, err := r.Render(ctx)
		if err != nil {
			t.Skip()
		}
		for _, key := range []string{"radiusd.conf", "clients.conf", "sites-enabled/default"} {
			if _, ok := files[key]; !ok {
				t.Fatalf("missing %s", key)
			}
		}
		for _, mod := range ctx.Cluster.Modules {
			if mod.Enabled {
				if _, ok := files["mods-enabled/"+mod.Name]; !ok {
					t.Fatalf("missing mods-enabled/%s", mod.Name)
				}
			}
		}
	})
}

func TestAllClientsInConfig(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		nClients := rapid.IntRange(0, 8).Draw(t, "nClients")
		clients := make([]ClientSpec, nClients)
		for i := range clients {
			clients[i] = genClientSpec(t)
		}
		ctx := RenderContext{Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"}, Clients: clients}
		files, err := r.Render(ctx)
		if err != nil {
			t.Fatal(err)
		}
		clientsConf := files["clients.conf"]
		for _, c := range clients {
			if !strings.Contains(clientsConf, "client "+c.Name+" {") {
				t.Fatalf("client %q not found in clients.conf", c.Name)
			}
		}
		// +2 for localhost + localhost_v6
		if count := strings.Count(clientsConf, "client "); count != nClients+2 {
			t.Fatalf("expected %d client blocks, got %d", nClients+2, count)
		}
	})
}

func TestPolicyPriorityOrdering(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		stage := rapid.SampledFrom(validStages).Draw(t, "stage")
		nPolicies := rapid.IntRange(2, 6).Draw(t, "nPolicies")
		policies := make([]PolicySpec, nPolicies)
		usedPriorities := make(map[int32]bool)
		for i := range policies {
			var priority int32
			for {
				priority = rapid.Int32Range(0, 10000).Draw(t, "priority")
				if !usedPriorities[priority] {
					break
				}
			}
			usedPriorities[priority] = true
			policies[i] = PolicySpec{
				Name:  rapid.StringMatching(`[a-z][a-z0-9]{3,10}`).Draw(t, "pname") + int32ToStr(int32(i)),
				Stage: stage, Priority: priority, Actions: []PolicyAction{{Type: "accept"}},
			}
		}
		ctx := RenderContext{Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"}, Policies: policies}
		files, err := r.Render(ctx)
		if err != nil {
			t.Fatal(err)
		}
		sitesDefault := files["sites-enabled/default"]
		stageHeader := "    " + stage + " {"
		stageStart := strings.Index(sitesDefault, stageHeader)
		if stageStart == -1 {
			t.Fatalf("stage %q not found", stage)
		}
		afterHeader := sitesDefault[stageStart+len(stageHeader):]
		endIdx := strings.Index(afterHeader, "\n    }")
		stageSection := afterHeader
		if endIdx != -1 {
			stageSection = afterHeader[:endIdx]
		}
		type pp struct {
			priority int32
			pos      int
			name     string
		}
		var positions []pp
		for _, p := range policies {
			pos := strings.Index(stageSection, p.Name)
			if pos == -1 {
				t.Fatalf("policy %q not found in stage %q", p.Name, stage)
			}
			positions = append(positions, pp{p.Priority, pos, p.Name})
		}
		for i := 0; i < len(positions); i++ {
			for j := i + 1; j < len(positions); j++ {
				if positions[i].priority < positions[j].priority && positions[i].pos > positions[j].pos {
					t.Fatalf("%q (pri %d) after %q (pri %d)", positions[i].name, positions[i].priority, positions[j].name, positions[j].priority)
				}
				if positions[i].priority > positions[j].priority && positions[i].pos < positions[j].pos {
					t.Fatalf("%q (pri %d) before %q (pri %d)", positions[i].name, positions[i].priority, positions[j].name, positions[j].priority)
				}
			}
		}
	})
}

func TestEnabledModulesRendered(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		nMods := rapid.IntRange(1, 6).Draw(t, "nMods")
		modules := make([]ModuleConfig, nMods)
		usedNames := make(map[string]bool)
		for i := range modules {
			mod := genModuleConfig(t)
			mod.Name = mod.Name + int32ToStr(int32(i))
			for usedNames[mod.Name] {
				mod.Name += "x"
			}
			usedNames[mod.Name] = true
			modules[i] = mod
		}
		ctx := RenderContext{Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3", Modules: modules}}
		files, err := r.Render(ctx)
		if err != nil {
			t.Skip()
		}
		for _, mod := range modules {
			_, exists := files["mods-enabled/"+mod.Name]
			if mod.Enabled && !exists {
				t.Fatalf("enabled module %q missing", mod.Name)
			}
			if !mod.Enabled && exists {
				t.Fatalf("disabled module %q present", mod.Name)
			}
		}
	})
}

func TestConfigRenderRoundTrip(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		ctx := genRenderContext(t)
		files1, err := r.Render(ctx)
		if err != nil {
			t.Skip()
		}
		files2, err := r.Render(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(files1) != len(files2) {
			t.Fatalf("file count mismatch: %d vs %d", len(files1), len(files2))
		}
		for k, v1 := range files1 {
			if v2, ok := files2[k]; !ok {
				t.Fatalf("key %q missing in second render", k)
			} else if normalizeWS(v1) != normalizeWS(v2) {
				t.Fatalf("content mismatch for %q", k)
			}
		}
	})
}

func TestSecretRefNotPlaintext(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		secretName := rapid.StringMatching(`[a-z][a-z0-9-]{2,15}`).Draw(t, "secretName")
		secretKey := rapid.StringMatching(`[a-z][a-z0-9-]{2,15}`).Draw(t, "secretKey")
		secretValue := rapid.StringMatching(`[A-Za-z0-9]{8,32}`).Draw(t, "secretValue")
		ref := SecretRef{Name: secretName, Key: secretKey}
		ctx := RenderContext{
			Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3", Modules: []ModuleConfig{{
				Name: "sql1", Type: "rlm_sql", Enabled: true,
				SQL: &SQLConfig{Dialect: "postgresql", Server: "db.example.com", Port: 5432, Database: "radius", Login: "radius", PasswordRef: ref},
			}}},
			Clients: []ClientSpec{{Name: "client1", IP: "10.0.0.1", SecretRef: ref, NASType: "other"}},
		}
		files, err := r.Render(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for filename, content := range files {
			if strings.Contains(content, secretValue) {
				t.Fatalf("plaintext secret found in %q", filename)
			}
		}
		expectedPath := "${file:/etc/freeradius/secrets/" + secretName + "/" + secretKey + "}"
		found := false
		for _, content := range files {
			if strings.Contains(content, expectedPath) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected file path ref %q not found", expectedPath)
		}
	})
}

func TestModuleRequiredFields(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		modType := rapid.SampledFrom([]string{"rlm_sql", "rlm_ldap", "rlm_eap", "rlm_redis"}).Draw(t, "modType")
		mod := ModuleConfig{Name: "testmod", Type: modType, Enabled: true}
		switch modType {
		case "rlm_sql":
			mod.SQL = genSQLConfig(t)
		case "rlm_ldap":
			mod.LDAP = genLDAPConfig(t)
		case "rlm_eap":
			mod.EAP = genEAPConfig(t)
		case "rlm_redis":
			mod.Redis = genRedisConfig(t)
		}
		ctx := RenderContext{Cluster: ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3", Modules: []ModuleConfig{mod}}}
		files, err := r.Render(ctx)
		if err != nil {
			t.Fatal(err)
		}
		content := files["mods-enabled/testmod"]
		required := map[string][]string{
			"rlm_sql":   {"driver", "dialect", "server", "port", "database", "login", "password"},
			"rlm_ldap":  {"server", "port", "base_dn", "identity", "password"},
			"rlm_eap":   {"default_eap_type"},
			"rlm_redis": {"server", "port"},
		}
		for _, field := range required[modType] {
			if !strings.Contains(content, field) {
				t.Fatalf("%s missing field %q", modType, field)
			}
		}
	})
}

func TestPolicyInCorrectStage(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		stage := rapid.SampledFrom(validStages).Draw(t, "stage")
		name := rapid.StringMatching(`[a-z][a-z0-9]{3,10}`).Draw(t, "policyName")
		ctx := RenderContext{
			Cluster:  ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"},
			Policies: []PolicySpec{{Name: name, Stage: stage, Priority: 10, Actions: []PolicyAction{{Type: "accept"}}}},
		}
		files, err := r.Render(ctx)
		if err != nil {
			t.Fatal(err)
		}
		sitesDefault := files["sites-enabled/default"]
		stageHeader := "    " + stage + " {"
		idx := strings.Index(sitesDefault, stageHeader)
		if idx == -1 {
			t.Fatalf("stage %q not found", stage)
		}
		afterHeader := sitesDefault[idx+len(stageHeader):]
		endIdx := strings.Index(afterHeader, "\n    }")
		section := afterHeader
		if endIdx != -1 {
			section = afterHeader[:endIdx]
		}
		if !strings.Contains(section, name) {
			t.Fatalf("policy %q not in stage %q", name, stage)
		}
		for _, other := range validStages {
			if other == stage {
				continue
			}
			oHeader := "    " + other + " {"
			oIdx := strings.Index(sitesDefault, oHeader)
			if oIdx == -1 {
				continue
			}
			afterO := sitesDefault[oIdx+len(oHeader):]
			oEnd := strings.Index(afterO, "\n    }")
			oSection := afterO
			if oEnd != -1 {
				oSection = afterO[:oEnd]
			}
			if strings.Contains(oSection, name) {
				t.Fatalf("policy %q found in wrong stage %q", name, other)
			}
		}
	})
}

func normalizeWS(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return strings.Join(lines, "\n")
}

func int32ToStr(n int32) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
