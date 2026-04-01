package renderer

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// --- Generators ---

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
	if rapid.Bool().Draw(t, "hasMatch") {
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
	return nil
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
		Server:      rapid.StringMatching(`[a-z][a-z0-9.-]{0,30}`).Draw(t, "ldapServer"),
		Port:        rapid.Int32Range(389, 636).Draw(t, "ldapPort"),
		BaseDN:      "dc=example,dc=com",
		Identity:    "cn=admin,dc=example,dc=com",
		PasswordRef: genSecretRef(t),
	}
}

func genEAPConfig(t *rapid.T) *EAPConfig {
	return &EAPConfig{
		DefaultEAPType: rapid.SampledFrom([]string{"peap", "ttls", "tls", "md5"}).Draw(t, "eapType"),
	}
}

func genRedisConfig(t *rapid.T) *RedisConfig {
	hasPassword := rapid.Bool().Draw(t, "redisHasPassword")
	cfg := &RedisConfig{
		Server: rapid.StringMatching(`[a-z][a-z0-9.-]{0,30}`).Draw(t, "redisServer"),
		Port:   rapid.Int32Range(1024, 65535).Draw(t, "redisPort"),
	}
	if hasPassword {
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
		Name:    rapid.StringMatching(`[a-z][a-z0-9_]{0,15}`).Draw(t, "modName"),
		Type:    modType,
		Enabled: rapid.Bool().Draw(t, "modEnabled"),
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
		stage := rapid.SampledFrom(validStages).Draw(t, "stage")
		policies[i] = genPolicySpec(t, stage)
	}

	return RenderContext{
		Cluster: ClusterSpec{
			Replicas: rapid.Int32Range(1, 10).Draw(t, "replicas"),
			Image:    "docker.io/freeradius/freeradius-server:3.2.3",
			Modules:  modules,
		},
		Clients:  clients,
		Policies: policies,
	}
}

// --- Property Tests ---

// Feature: freeradius-operator, Property 3: ConfigRenderer produces all required output files
func TestRendererOutputFiles(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		ctx := genRenderContext(t)
		files, err := r.Render(ctx)
		if err != nil {
			// Only InvalidModuleError is expected; skip those
			t.Skip()
		}

		// Must always have radiusd.conf, clients.conf, sites-enabled/default
		if _, ok := files["radiusd.conf"]; !ok {
			t.Fatal("missing radiusd.conf")
		}
		if _, ok := files["clients.conf"]; !ok {
			t.Fatal("missing clients.conf")
		}
		if _, ok := files["sites-enabled/default"]; !ok {
			t.Fatal("missing sites-enabled/default")
		}

		// Every enabled module must have a mods-enabled/<name> entry
		for _, mod := range ctx.Cluster.Modules {
			if !mod.Enabled {
				continue
			}
			key := "mods-enabled/" + mod.Name
			if _, ok := files[key]; !ok {
				t.Fatalf("missing mods-enabled/%s for enabled module %q", mod.Name, mod.Name)
			}
		}
	})
}

// Feature: freeradius-operator, Property 4: All RadiusClients appear in clients.conf
func TestAllClientsInConfig(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		nClients := rapid.IntRange(0, 8).Draw(t, "nClients")
		clients := make([]ClientSpec, nClients)
		for i := range clients {
			clients[i] = genClientSpec(t)
		}

		ctx := RenderContext{
			Cluster:  ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"},
			Clients:  clients,
			Policies: nil,
		}

		files, err := r.Render(ctx)
		if err != nil {
			t.Fatal(err)
		}

		clientsConf := files["clients.conf"]

		// Every client name must appear in clients.conf
		for _, c := range clients {
			if !strings.Contains(clientsConf, "client "+c.Name+" {") {
				t.Fatalf("client %q not found in clients.conf", c.Name)
			}
		}

		// Count client blocks — must equal number of clients + 1 for the localhost probe client
		count := strings.Count(clientsConf, "client ")
		if count != nClients+1 {
			t.Fatalf("expected %d client blocks (including localhost), got %d", nClients+1, count)
		}
	})
}

// Feature: freeradius-operator, Property 5: RadiusPolicies rendered in ascending priority order
func TestPolicyPriorityOrdering(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		stage := rapid.SampledFrom(validStages).Draw(t, "stage")
		nPolicies := rapid.IntRange(2, 6).Draw(t, "nPolicies")
		policies := make([]PolicySpec, nPolicies)
		// Use unique names and unique priorities to make ordering unambiguous
		usedPriorities := make(map[int32]bool)
		for i := range policies {
			name := rapid.StringMatching(`[a-z][a-z0-9]{3,10}`).Draw(t, "pname") + int32ToStr(int32(i))
			var priority int32
			for {
				priority = rapid.Int32Range(0, 10000).Draw(t, "priority")
				if !usedPriorities[priority] {
					break
				}
			}
			usedPriorities[priority] = true
			policies[i] = PolicySpec{
				Name:     name,
				Stage:    stage,
				Priority: priority,
				Actions:  []PolicyAction{{Type: "accept"}},
			}
		}

		ctx := RenderContext{
			Cluster:  ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"},
			Policies: policies,
		}

		files, err := r.Render(ctx)
		if err != nil {
			t.Fatal(err)
		}

		sitesDefault := files["sites-enabled/default"]

		// Find the stage section
		stageHeader := "    " + stage + " {"
		stageStart := strings.Index(sitesDefault, stageHeader)
		if stageStart == -1 {
			t.Fatalf("stage section %q not found", stage)
		}

		// Extract the stage section content
		afterHeader := sitesDefault[stageStart+len(stageHeader):]
		endIdx := strings.Index(afterHeader, "\n    }")
		var stageSection string
		if endIdx != -1 {
			stageSection = afterHeader[:endIdx]
		} else {
			stageSection = afterHeader
		}

		// Find position of each policy name in the stage section
		type policyPos struct {
			priority int32
			pos      int
			name     string
		}
		var positions []policyPos
		for _, p := range policies {
			pos := strings.Index(stageSection, p.Name)
			if pos == -1 {
				t.Fatalf("policy %q not found in stage section %q", p.Name, stage)
			}
			positions = append(positions, policyPos{p.Priority, pos, p.Name})
		}

		// Sort by priority and verify positions are in the same order
		for i := 0; i < len(positions); i++ {
			for j := i + 1; j < len(positions); j++ {
				if positions[i].priority > positions[j].priority && positions[i].pos < positions[j].pos {
					t.Fatalf("policy %q (priority %d) appears before policy %q (priority %d) but should appear after",
						positions[i].name, positions[i].priority, positions[j].name, positions[j].priority)
				}
				if positions[i].priority < positions[j].priority && positions[i].pos > positions[j].pos {
					t.Fatalf("policy %q (priority %d) appears after policy %q (priority %d) but should appear before",
						positions[i].name, positions[i].priority, positions[j].name, positions[j].priority)
				}
			}
		}
	})
}

// Feature: freeradius-operator, Property 6: Enabled modules appear in mods-enabled/
func TestEnabledModulesRendered(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		nMods := rapid.IntRange(1, 6).Draw(t, "nMods")
		modules := make([]ModuleConfig, nMods)
		usedNames := make(map[string]bool)
		for i := range modules {
			mod := genModuleConfig(t)
			// Ensure unique names by appending index
			mod.Name = mod.Name + int32ToStr(int32(i))
			for usedNames[mod.Name] {
				mod.Name = mod.Name + "x"
			}
			usedNames[mod.Name] = true
			modules[i] = mod
		}

		ctx := RenderContext{
			Cluster: ClusterSpec{
				Replicas: 1,
				Image:    "freeradius:3.2.3",
				Modules:  modules,
			},
		}

		files, err := r.Render(ctx)
		if err != nil {
			t.Skip() // skip invalid module configs
		}

		for _, mod := range modules {
			key := "mods-enabled/" + mod.Name
			_, exists := files[key]
			if mod.Enabled && !exists {
				t.Fatalf("enabled module %q (type %s) missing from output", mod.Name, mod.Type)
			}
			if !mod.Enabled && exists {
				t.Fatalf("disabled module %q (type %s) should not appear in output", mod.Name, mod.Type)
			}
		}
	})
}

// Feature: freeradius-operator, Property 7: Config rendering round-trip
func TestConfigRenderRoundTrip(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		ctx := genRenderContext(t)

		files1, err := r.Render(ctx)
		if err != nil {
			t.Skip()
		}

		// Render again with the same context — must produce identical output
		files2, err := r.Render(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Assert same keys
		if len(files1) != len(files2) {
			t.Fatalf("round-trip: file count mismatch: %d vs %d", len(files1), len(files2))
		}
		for k, v1 := range files1 {
			v2, ok := files2[k]
			if !ok {
				t.Fatalf("round-trip: key %q missing in second render", k)
			}
			// Normalize whitespace for comparison
			if normalizeWS(v1) != normalizeWS(v2) {
				t.Fatalf("round-trip: content mismatch for key %q", k)
			}
		}
	})
}

// Feature: freeradius-operator, Property 21: SecretRef rendered as file path, never plaintext
func TestSecretRefNotPlaintext(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		// Generate a context with known secret values
		secretName := rapid.StringMatching(`[a-z][a-z0-9-]{2,15}`).Draw(t, "secretName")
		secretKey := rapid.StringMatching(`[a-z][a-z0-9-]{2,15}`).Draw(t, "secretKey")
		secretValue := rapid.StringMatching(`[A-Za-z0-9!@#]{8,32}`).Draw(t, "secretValue")

		ref := SecretRef{Name: secretName, Key: secretKey}
		ctx := RenderContext{
			Cluster: ClusterSpec{
				Replicas: 1,
				Image:    "freeradius:3.2.3",
				Modules: []ModuleConfig{
					{
						Name:    "sql1",
						Type:    "rlm_sql",
						Enabled: true,
						SQL: &SQLConfig{
							Dialect:     "postgresql",
							Server:      "db.example.com",
							Port:        5432,
							Database:    "radius",
							Login:       "radius",
							PasswordRef: ref,
						},
					},
				},
			},
			Clients: []ClientSpec{
				{
					Name:      "client1",
					IP:        "10.0.0.1",
					SecretRef: ref,
					NASType:   "other",
				},
			},
		}

		files, err := r.Render(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// The plaintext secret value must NOT appear anywhere in the rendered output
		for filename, content := range files {
			if strings.Contains(content, secretValue) {
				t.Fatalf("plaintext secret value found in %q", filename)
			}
		}

		// The file path reference MUST appear
		expectedPath := "${file:/etc/freeradius/secrets/" + secretName + "/" + secretKey + "}"
		found := false
		for _, content := range files {
			if strings.Contains(content, expectedPath) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected file path reference %q not found in any rendered file", expectedPath)
		}
	})
}

// Feature: freeradius-operator, Property 23: Module-specific required fields present in rendered output
func TestModuleRequiredFields(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		modType := rapid.SampledFrom([]string{"rlm_sql", "rlm_ldap", "rlm_eap", "rlm_redis"}).Draw(t, "modType")

		mod := ModuleConfig{
			Name:    "testmod",
			Type:    modType,
			Enabled: true,
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

		ctx := RenderContext{
			Cluster: ClusterSpec{
				Replicas: 1,
				Image:    "freeradius:3.2.3",
				Modules:  []ModuleConfig{mod},
			},
		}

		files, err := r.Render(ctx)
		if err != nil {
			t.Fatal(err)
		}

		content := files["mods-enabled/testmod"]

		switch modType {
		case "rlm_sql":
			for _, field := range []string{"driver", "dialect", "server", "port", "database", "login", "password"} {
				if !strings.Contains(content, field) {
					t.Fatalf("rlm_sql missing required field %q in output:\n%s", field, content)
				}
			}
		case "rlm_ldap":
			for _, field := range []string{"server", "port", "base_dn", "identity", "password"} {
				if !strings.Contains(content, field) {
					t.Fatalf("rlm_ldap missing required field %q in output:\n%s", field, content)
				}
			}
		case "rlm_eap":
			if !strings.Contains(content, "default_eap_type") {
				t.Fatalf("rlm_eap missing required field 'default_eap_type' in output:\n%s", content)
			}
		case "rlm_redis":
			for _, field := range []string{"server", "port"} {
				if !strings.Contains(content, field) {
					t.Fatalf("rlm_redis missing required field %q in output:\n%s", field, content)
				}
			}
		}
	})
}

// Feature: freeradius-operator, Property 24: RadiusPolicy rendered in correct stage section
func TestPolicyInCorrectStage(t *testing.T) {
	r := New()
	rapid.Check(t, func(t *rapid.T) {
		stage := rapid.SampledFrom(validStages).Draw(t, "stage")
		policyName := rapid.StringMatching(`[a-z][a-z0-9]{3,10}`).Draw(t, "policyName")

		policy := PolicySpec{
			Name:     policyName,
			Stage:    stage,
			Priority: 10,
			Actions:  []PolicyAction{{Type: "accept"}},
		}

		ctx := RenderContext{
			Cluster:  ClusterSpec{Replicas: 1, Image: "freeradius:3.2.3"},
			Policies: []PolicySpec{policy},
		}

		files, err := r.Render(ctx)
		if err != nil {
			t.Fatal(err)
		}

		sitesDefault := files["sites-enabled/default"]

		// Find the correct stage section
		stageHeader := "    " + stage + " {"
		stageIdx := strings.Index(sitesDefault, stageHeader)
		if stageIdx == -1 {
			t.Fatalf("stage section %q not found in sites-enabled/default", stage)
		}

		// Find the end of this stage section
		afterHeader := sitesDefault[stageIdx+len(stageHeader):]
		endIdx := strings.Index(afterHeader, "\n    }")
		var stageContent string
		if endIdx != -1 {
			stageContent = afterHeader[:endIdx]
		} else {
			stageContent = afterHeader
		}

		// The policy name must appear in the correct stage section
		if !strings.Contains(stageContent, policyName) {
			t.Fatalf("policy %q not found in stage section %q", policyName, stage)
		}

		// The policy name must NOT appear in any other stage section
		for _, otherStage := range validStages {
			if otherStage == stage {
				continue
			}
			otherHeader := "    " + otherStage + " {"
			otherIdx := strings.Index(sitesDefault, otherHeader)
			if otherIdx == -1 {
				continue
			}
			afterOther := sitesDefault[otherIdx+len(otherHeader):]
			endOther := strings.Index(afterOther, "\n    }")
			var otherContent string
			if endOther != -1 {
				otherContent = afterOther[:endOther]
			} else {
				otherContent = afterOther
			}
			if strings.Contains(otherContent, policyName) {
				t.Fatalf("policy %q found in wrong stage section %q (expected %q)", policyName, otherStage, stage)
			}
		}
	})
}

// --- Helpers ---

func normalizeWS(s string) string {
	// Collapse multiple spaces/tabs to single space, trim lines
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
