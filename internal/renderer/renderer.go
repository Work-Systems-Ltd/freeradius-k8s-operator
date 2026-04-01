// Package renderer implements the ConfigRenderer interface for generating
// FreeRADIUS configuration files from CRD specs.
package renderer

import "fmt"

// ConfigFiles maps relative file paths to rendered content.
// Keys: "radiusd.conf", "clients.conf", "mods-enabled/<name>", "sites-enabled/default"
type ConfigFiles map[string]string

// SecretRef references a key within a Kubernetes Secret.
type SecretRef struct {
	Name string
	Key  string
}

// SQLConfig holds configuration for the rlm_sql module.
type SQLConfig struct {
	Dialect     string
	Server      string
	Port        int32
	Database    string
	Login       string
	PasswordRef SecretRef
}

// LDAPConfig holds configuration for the rlm_ldap module.
type LDAPConfig struct {
	Server      string
	Port        int32
	BaseDN      string
	Identity    string
	PasswordRef SecretRef
}

// EAPTLSConfig holds TLS sub-type configuration for EAP.
type EAPTLSConfig struct {
	CertFile string
	KeyFile  string
}

// EAPTTLSConfig holds TTLS sub-type configuration for EAP.
type EAPTTLSConfig struct {
	DefaultEAPType string
	VirtualServer  string
}

// EAPPEAPConfig holds PEAP sub-type configuration for EAP.
type EAPPEAPConfig struct {
	DefaultEAPType string
	VirtualServer  string
}

// EAPConfig holds configuration for the rlm_eap module.
type EAPConfig struct {
	DefaultEAPType string
	TLS            *EAPTLSConfig
	TTLS           *EAPTTLSConfig
	PEAP           *EAPPEAPConfig
}

// RESTStageConfig holds per-stage URI/method overrides for the rlm_rest module.
type RESTStageConfig struct {
	URI    string
	Method string
}

// RESTConfig holds configuration for the rlm_rest module.
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

// RedisConfig holds configuration for the rlm_redis module.
type RedisConfig struct {
	Server      string
	Port        int32
	Database    int32
	PasswordRef *SecretRef
}

// ModuleConfig declares a single RLM backend instance.
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

// ClusterSpec mirrors RadiusClusterSpec but is decoupled from Kubernetes API types.
type ClusterSpec struct {
	Replicas int32
	Image    string
	Modules  []ModuleConfig
}

// ClientSpec mirrors RadiusClientSpec but is decoupled from Kubernetes API types.
type ClientSpec struct {
	Name      string
	IP        string
	SecretRef SecretRef
	NASType   string
}

// MatchLeaf is a single RADIUS attribute condition.
type MatchLeaf struct {
	Attribute string
	Operator  string
	Value     string
}

// PolicyMatch is a condition tree node supporting all/any/none combinators.
type PolicyMatch struct {
	All  []MatchLeaf
	Any  []MatchLeaf
	None []MatchLeaf
}

// PolicyAction is a single unlang action.
type PolicyAction struct {
	Type      string // set|call|reject|accept
	Module    string
	Attribute string
	Value     string
}

// PolicySpec mirrors RadiusPolicySpec but is decoupled from Kubernetes API types.
type PolicySpec struct {
	Name     string
	Stage    string
	Priority int32
	Match    *PolicyMatch
	Actions  []PolicyAction
}

// RenderContext holds all inputs needed to render FreeRADIUS config files.
type RenderContext struct {
	Cluster  ClusterSpec
	Clients  []ClientSpec
	Policies []PolicySpec
}

// InvalidModuleError is returned when an unrecognized module type is encountered.
type InvalidModuleError struct {
	ModuleType string
}

func (e *InvalidModuleError) Error() string {
	return "unrecognized module type: " + e.ModuleType
}

// InvalidStageError is returned when an unrecognized stage is encountered.
type InvalidStageError struct {
	Stage string
}

func (e *InvalidStageError) Error() string {
	return "unrecognized stage: " + e.Stage
}

// InvalidActionError is returned when an unrecognized action type is encountered.
type InvalidActionError struct {
	ActionType string
}

func (e *InvalidActionError) Error() string {
	return "unrecognized action type: " + e.ActionType
}

// ConfigRenderer generates FreeRADIUS config files from CRD specs.
// It is a pure function — no side effects, no API calls.
type ConfigRenderer interface {
	Render(ctx RenderContext) (ConfigFiles, error)
}

// New returns a new ConfigRenderer.
func New() ConfigRenderer {
	return &defaultRenderer{}
}

type defaultRenderer struct{}

// Render generates all FreeRADIUS configuration files from the given RenderContext.
func (r *defaultRenderer) Render(ctx RenderContext) (ConfigFiles, error) {
	// Validate all inputs before rendering to prevent config injection.
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

	// radiusd.conf
	radiusd, err := renderRadiusd(ctx.Cluster)
	if err != nil {
		return nil, err
	}
	files["radiusd.conf"] = radiusd

	// clients.conf
	clients, err := renderClients(ctx.Clients)
	if err != nil {
		return nil, err
	}
	files["clients.conf"] = clients

	// mods-enabled/<name>
	modFiles, err := renderModules(ctx.Cluster.Modules)
	if err != nil {
		return nil, err
	}
	for k, v := range modFiles {
		files[k] = v
	}

	// sites-enabled/default
	sites, err := renderSites(ctx.Policies)
	if err != nil {
		return nil, err
	}
	files["sites-enabled/default"] = sites

	return files, nil
}
