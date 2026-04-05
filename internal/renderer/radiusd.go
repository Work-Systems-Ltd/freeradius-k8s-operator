package renderer

import "bytes"

func renderRadiusd(cluster ClusterSpec) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, tmplRadiusd, cluster); err != nil {
		return "", err
	}
	return buf.String(), nil
}
