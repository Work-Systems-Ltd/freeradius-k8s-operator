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

func yesno(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

var funcMap = template.FuncMap{
	"secretFilePath":    secretFilePath,
	"secretFilePathPtr": secretFilePathPtr,
	"ipaddrDirective":   ipaddrDirective,
	"condition":         buildCondition,
	"renderAction":      renderActionStr,
	"yesno":             yesno,
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
	case "redundant", "load-balance":
		return renderGroupAction(action.Type, action.Modules, indent)
	default:
		return ""
	}
}

func renderGroupAction(keyword string, modules []string, indent string) string {
	var b strings.Builder
	b.WriteString(indent + keyword + " {\n")
	for _, m := range modules {
		b.WriteString(indent + "    " + m + "\n")
	}
	b.WriteString(indent + "}\n")
	return b.String()
}
