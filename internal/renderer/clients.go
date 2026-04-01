package renderer

import "bytes"

// RenderClients renders a clients.conf file for the given client specs.
// Exported so the render-clients init container binary can use it.
func RenderClients(clients []ClientSpec) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "clients.conf.tmpl", clients); err != nil {
		return "", err
	}
	return buf.String(), nil
}
