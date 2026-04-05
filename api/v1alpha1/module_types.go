// +kubebuilder:object:generate=true
package v1alpha1

type SQLConfig struct {
	// +kubebuilder:validation:Enum=mysql;postgresql;sqlite;mssql;oracle;mongo
	Dialect     string    `json:"dialect"`
	Server      string    `json:"server"`
	Port        int32     `json:"port"`
	Database    string    `json:"database"`
	Login       string    `json:"login"`
	PasswordRef SecretRef `json:"passwordRef"`
}

type LDAPConfig struct {
	Server      string    `json:"server"`
	Port        int32     `json:"port"`
	BaseDN      string    `json:"baseDN"`
	Identity    string    `json:"identity"`
	PasswordRef SecretRef `json:"passwordRef"`
}

type EAPTLSConfig struct {
	CertFile string `json:"certFile,omitempty"`
	KeyFile  string `json:"keyFile,omitempty"`
}

type EAPTTLSConfig struct {
	DefaultEAPType string `json:"defaultEAPType,omitempty"`
	VirtualServer  string `json:"virtualServer,omitempty"`
}

type EAPPEAPConfig struct {
	DefaultEAPType string `json:"defaultEAPType,omitempty"`
	VirtualServer  string `json:"virtualServer,omitempty"`
}

type EAPConfig struct {
	DefaultEAPType string         `json:"defaultEAPType,omitempty"`
	TLS            *EAPTLSConfig  `json:"tls,omitempty"`
	TTLS           *EAPTTLSConfig `json:"ttls,omitempty"`
	PEAP           *EAPPEAPConfig `json:"peap,omitempty"`
}

type RESTStageConfig struct {
	URI    string `json:"uri,omitempty"`
	Method string `json:"method,omitempty"`
}

type RESTConfig struct {
	ConnectURI   string           `json:"connectURI"`
	Auth         string           `json:"auth,omitempty"`
	PasswordRef  *SecretRef       `json:"passwordRef,omitempty"`
	Authorize    *RESTStageConfig `json:"authorize,omitempty"`
	Authenticate *RESTStageConfig `json:"authenticate,omitempty"`
	Preacct      *RESTStageConfig `json:"preacct,omitempty"`
	Accounting   *RESTStageConfig `json:"accounting,omitempty"`
	PostAuth     *RESTStageConfig `json:"postAuth,omitempty"`
	PreProxy     *RESTStageConfig `json:"preProxy,omitempty"`
	PostProxy    *RESTStageConfig `json:"postProxy,omitempty"`
}

type RedisConfig struct {
	Server      string     `json:"server"`
	Port        int32      `json:"port"`
	Database    int32      `json:"database,omitempty"`
	PasswordRef *SecretRef `json:"passwordRef,omitempty"`
}

type FilesConfig struct {
	// Key is the RADIUS attribute used for user lookup.
	// Defaults to FreeRADIUS default if empty.
	Key string `json:"key,omitempty"`
	// Authorize contains Livingston-style user entries for authentication.
	Authorize string `json:"authorize,omitempty"`
	// Accounting contains user entries for accounting rules.
	Accounting string `json:"accounting,omitempty"`
}

type ModuleConfig struct {
	Name      string       `json:"name"`
	Type      string       `json:"type"`
	Enabled   bool         `json:"enabled"`
	SQL       *SQLConfig   `json:"sql,omitempty"`
	LDAP      *LDAPConfig  `json:"ldap,omitempty"`
	EAP       *EAPConfig   `json:"eap,omitempty"`
	REST      *RESTConfig  `json:"rest,omitempty"`
	Redis     *RedisConfig `json:"redis,omitempty"`
	Files     *FilesConfig `json:"files,omitempty"`
	RawConfig string       `json:"rawConfig,omitempty"`
}
