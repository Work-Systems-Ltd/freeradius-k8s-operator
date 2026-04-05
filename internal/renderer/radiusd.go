package renderer

import "bytes"

type radiusdContext struct {
	MaxRequestTime int32
	MaxRequests    int32
	Log            RadiusdLogConfig
	Security       RadiusdSecurityConfig
	ThreadPool     RadiusdThreadPool
	CoAEnabled     bool
	CoAPort        int32
}

func boolDefault(v bool, def bool) bool {
	return v || def
}

func int32Default(v, def int32) int32 {
	if v != 0 {
		return v
	}
	return def
}

func stringDefault(v, def string) string {
	if v != "" {
		return v
	}
	return def
}

func buildRadiusdContext(cluster ClusterSpec) radiusdContext {
	r := cluster.Radiusd
	return radiusdContext{
		MaxRequestTime: int32Default(r.MaxRequestTime, 30),
		MaxRequests:    int32Default(r.MaxRequests, 16384),
		Log: RadiusdLogConfig{
			Destination:  stringDefault(r.Log.Destination, "stdout"),
			Auth:         boolDefault(r.Log.Auth, true),
			AuthBadpass:  r.Log.AuthBadpass,
			AuthGoodpass: r.Log.AuthGoodpass,
		},
		Security: RadiusdSecurityConfig{
			MaxAttributes: int32Default(r.Security.MaxAttributes, 200),
			RejectDelay:   int32Default(r.Security.RejectDelay, 1),
		},
		ThreadPool: RadiusdThreadPool{
			StartServers:         int32Default(r.ThreadPool.StartServers, 5),
			MaxServers:           int32Default(r.ThreadPool.MaxServers, 32),
			MinSpareServers:      int32Default(r.ThreadPool.MinSpareServers, 3),
			MaxSpareServers:      int32Default(r.ThreadPool.MaxSpareServers, 10),
			MaxRequestsPerServer: r.ThreadPool.MaxRequestsPerServer,
		},
		CoAEnabled: cluster.CoAEnabled,
		CoAPort:    cluster.CoAPort,
	}
}

func renderRadiusd(cluster ClusterSpec) (string, error) {
	if cluster.Radiusd.RawConfig != "" {
		return cluster.Radiusd.RawConfig, nil
	}

	ctx := buildRadiusdContext(cluster)
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, tmplRadiusd, ctx); err != nil {
		return "", err
	}
	return buf.String(), nil
}
