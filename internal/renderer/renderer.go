package renderer

import "fmt"

type ConfigFiles map[string]string

type SecretRef struct {
	Name string
	Key  string
}

type SQLConfig struct {
	Dialect     string
	Server      string
	Port        int32
	Database    string
	Login       string
	PasswordRef SecretRef
}

type LDAPConfig struct {
	Server      string
	Port        int32
	BaseDN      string
	Identity    string
	PasswordRef SecretRef
}

type EAPTLSConfig struct {
	CertFile string
	KeyFile  string
}

type EAPTTLSConfig struct {
	DefaultEAPType string
	VirtualServer  string
}

type EAPPEAPConfig struct {
	DefaultEAPType string
	VirtualServer  string
}

type EAPConfig struct {
	DefaultEAPType string
	TLS            *EAPTLSConfig
	TTLS           *EAPTTLSConfig
	PEAP           *EAPPEAPConfig
}

type RESTStageConfig struct {
	URI    string
	Method string
}

type RESTConfig struct {
	ConnectURI   string
	Auth         string
	PasswordRef  *SecretRef
	Authorize    *RESTStageConfig
	Authenticate *RESTStageConfig
	Preacct      *RESTStageConfig
	Accounting   *RESTStageConfig
	PostAuth     *RESTStageConfig
	PreProxy     *RESTStageConfig
	PostProxy    *RESTStageConfig
}

type RedisConfig struct {
	Server      string
	Port        int32
	Database    int32
	PasswordRef *SecretRef
}

type ModuleConfig struct {
	Name    string
	Type    string
	Enabled bool
	SQL     *SQLConfig
	LDAP    *LDAPConfig
	EAP     *EAPConfig
	REST    *RESTConfig
	Redis   *RedisConfig
}

type ClusterSpec struct {
	Replicas   int32
	Image      string
	Modules    []ModuleConfig
	CoAEnabled bool
	CoAPort    int32
}

type ClientSpec struct {
	Name      string
	IP        string
	SecretRef SecretRef
	NASType   string
}

type MatchLeaf struct {
	Attribute string
	Operator  string
	Value     string
}

type PolicyMatch struct {
	All  []MatchLeaf
	Any  []MatchLeaf
	None []MatchLeaf
}

type PolicyAction struct {
	Type      string
	Module    string
	Attribute string
	Value     string
}

type PolicySpec struct {
	Name     string
	Stage    string
	Priority int32
	Match    *PolicyMatch
	Actions  []PolicyAction
}

type RenderContext struct {
	Cluster  ClusterSpec
	Clients  []ClientSpec
	Policies []PolicySpec
}

type InvalidModuleError struct{ ModuleType string }

func (e *InvalidModuleError) Error() string { return "unrecognized module type: " + e.ModuleType }

type InvalidStageError struct{ Stage string }

func (e *InvalidStageError) Error() string { return "unrecognized stage: " + e.Stage }

type InvalidActionError struct{ ActionType string }

func (e *InvalidActionError) Error() string { return "unrecognized action type: " + e.ActionType }

type ConfigRenderer interface {
	Render(ctx RenderContext) (ConfigFiles, error)
}

func New() ConfigRenderer { return &defaultRenderer{} }

type defaultRenderer struct{}

func (r *defaultRenderer) Render(ctx RenderContext) (ConfigFiles, error) {
	for _, c := range ctx.Clients {
		if err := validateClientSpec(c); err != nil {
			return nil, fmt.Errorf("validating client %q: %w", c.Name, err)
		}
	}
	for _, m := range ctx.Cluster.Modules {
		if m.Enabled {
			if err := validateModuleConfig(m); err != nil {
				return nil, fmt.Errorf("validating module %q: %w", m.Name, err)
			}
		}
	}
	for _, p := range ctx.Policies {
		if err := validatePolicySpec(p); err != nil {
			return nil, err
		}
	}

	files := make(ConfigFiles)

	// Shard clients across multiple files
	shards, err := shardClients(ctx.Clients, defaultShardMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("sharding clients: %w", err)
	}

	for i, shard := range shards {
		rendered, err := renderClientShard(shard)
		if err != nil {
			return nil, fmt.Errorf("rendering client shard %d: %w", i, err)
		}
		files[fmt.Sprintf("clients_%03d.conf", i)] = rendered
	}

	clients, err := renderClients(ctx.Clients, len(shards))
	if err != nil {
		return nil, err
	}
	files["clients.conf"] = clients

	radiusd, err := renderRadiusd(ctx.Cluster)
	if err != nil {
		return nil, err
	}
	files["radiusd.conf"] = radiusd

	modFiles, err := renderModules(ctx.Cluster.Modules)
	if err != nil {
		return nil, err
	}
	for k, v := range modFiles {
		files[k] = v
	}

	sites, err := renderSites(ctx.Policies, ctx.Cluster.CoAEnabled)
	if err != nil {
		return nil, err
	}
	files["sites-enabled/default"] = sites

	return files, nil
}
