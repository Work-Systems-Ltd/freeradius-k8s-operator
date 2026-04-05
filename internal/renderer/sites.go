package renderer

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

var validStages = []string{
	"authorize", "authenticate", "preacct", "accounting",
	"post-auth", "pre-proxy", "post-proxy", "session",
}

var validStageSet = func() map[string]bool {
	m := make(map[string]bool, len(validStages))
	for _, s := range validStages {
		m[s] = true
	}
	return m
}()

type stageData struct {
	Name     string
	Policies []PolicySpec
}

type sitesContext struct {
	Stages     []stageData
	CoAEnabled bool
}

func renderSites(policies []PolicySpec, coaEnabled ...bool) (string, error) {
	for _, p := range policies {
		if !validStageSet[p.Stage] {
			return "", &InvalidStageError{Stage: p.Stage}
		}
		for _, a := range p.Actions {
			switch a.Type {
			case "set", "call", "reject", "accept":
			default:
				return "", &InvalidActionError{ActionType: a.Type}
			}
		}
	}

	byStage := make(map[string][]PolicySpec)
	for _, p := range policies {
		byStage[p.Stage] = append(byStage[p.Stage], p)
	}
	for stage := range byStage {
		sort.Slice(byStage[stage], func(i, j int) bool {
			return byStage[stage][i].Priority < byStage[stage][j].Priority
		})
	}

	ctx := sitesContext{Stages: make([]stageData, len(validStages))}
	for i, stage := range validStages {
		ctx.Stages[i] = stageData{Name: stage, Policies: byStage[stage]}
	}
	if len(coaEnabled) > 0 && coaEnabled[0] {
		ctx.CoAEnabled = true
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, tmplDefault, ctx); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func buildCondition(match *PolicyMatch) string {
	if match == nil {
		return "true"
	}

	var groups []string

	if len(match.All) > 0 {
		groups = append(groups, joinLeaves(match.All, " && ", false))
	}
	if len(match.Any) > 0 {
		groups = append(groups, joinLeaves(match.Any, " || ", false))
	}
	if len(match.None) > 0 {
		groups = append(groups, joinLeaves(match.None, " && ", true))
	}

	if len(groups) == 0 {
		return "true"
	}
	if len(groups) == 1 {
		return groups[0]
	}
	return strings.Join(groups, " && ")
}

func joinLeaves(leaves []MatchLeaf, sep string, negate bool) string {
	wrapped := make([]string, len(leaves))
	for i, leaf := range leaves {
		val := leaf.Value
		if leaf.Operator == "=~" || leaf.Operator == "!~" {
			val = "/" + val + "/"
		}
		expr := fmt.Sprintf("(%s %s %s)", leaf.Attribute, leaf.Operator, val)
		if negate {
			expr = "!" + expr
		}
		wrapped[i] = expr
	}
	if len(wrapped) == 1 {
		return wrapped[0]
	}
	return "(" + strings.Join(wrapped, sep) + ")"
}
