package renderer

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates/core/*.tmpl templates/mods-enabled/*.tmpl templates/sites-enabled/*.tmpl
var templateFS embed.FS

func secretFilePath(ref SecretRef) string {
	return fmt.Sprintf("${file:/etc/freeradius/secrets/%s/%s}", ref.Name, ref.Key)
}

func secretFilePathPtr(ref *SecretRef) string {
	if ref == nil {
		return ""
	}
	return secretFilePath(*ref)
}

func ipaddrDirective(ip string) string {
	if strings.Contains(ip, ":") {
		return "ipv6addr"
	}
	return "ipaddr"
}

var funcMap = template.FuncMap{
	"secretFilePath":    secretFilePath,
	"secretFilePathPtr": secretFilePathPtr,
	"ipaddrDirective":   ipaddrDirective,
	"condition":         buildCondition,
	"renderAction":      renderActionStr,
}

var tmpl = template.Must(
	template.New("").Funcs(funcMap).ParseFS(templateFS,
		"templates/core/*.tmpl",
		"templates/mods-enabled/*.tmpl",
		"templates/sites-enabled/*.tmpl",
	),
)

func renderActionStr(action PolicyAction, indent string) string {
	switch action.Type {
	case "set":
		return fmt.Sprintf("%supdate reply {\n%s    %s := \"%s\"\n%s}\n", indent, indent, action.Attribute, action.Value, indent)
	case "call":
		return indent + action.Module + "\n"
	case "reject":
		return indent + "reject\n"
	case "accept":
		return indent + "ok\n"
	default:
		return ""
	}
}
