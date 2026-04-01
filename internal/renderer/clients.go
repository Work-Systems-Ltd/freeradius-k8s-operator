package renderer

import "bytes"

func renderClients(clients []ClientSpec) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "clients.conf.tmpl", clients); err != nil {
		return "", err
	}
	return buf.String(), nil
}
