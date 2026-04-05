package renderer

import "bytes"

func RenderClients(clients []ClientSpec) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, tmplClients, clients); err != nil {
		return "", err
	}
	return buf.String(), nil
}
