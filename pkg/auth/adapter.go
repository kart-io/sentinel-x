package auth

import (
	"context"
	"strings"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
)

// Adapter implements the persist.Adapter interface
type Adapter struct {
	repo Repository
}

// NewAdapter creates a new Adapter
func NewAdapter(repo Repository) *Adapter {
	return &Adapter{repo: repo}
}

// LoadPolicy loads all policy rules from the storage.
func (a *Adapter) LoadPolicy(model model.Model) error {
	policies, err := a.repo.LoadPolicies(context.Background())
	if err != nil {
		return err
	}

	for _, p := range policies {
		_ = persist.LoadPolicyLine(policyToString(p), model)
	}
	return nil
}

// SavePolicy saves all policy rules to the storage.
func (a *Adapter) SavePolicy(model model.Model) error {
	var policies []*Policy

	for ptype, ast := range model["p"] {
		for _, rule := range ast.Policy {
			policies = append(policies, stringToPolicy(ptype, rule))
		}
	}

	for ptype, ast := range model["g"] {
		for _, rule := range ast.Policy {
			policies = append(policies, stringToPolicy(ptype, rule))
		}
	}

	return a.repo.SavePolicies(context.Background(), policies)
}

// AddPolicy adds a policy rule to the storage.
func (a *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	return a.repo.AddPolicy(context.Background(), stringToPolicy(ptype, rule))
}

// RemovePolicy removes a policy rule from the storage.
func (a *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return a.repo.RemovePolicy(context.Background(), stringToPolicy(ptype, rule))
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return a.repo.RemoveFilteredPolicy(context.Background(), ptype, fieldIndex, fieldValues...)
}

func policyToString(p *Policy) string {
	var sb strings.Builder
	sb.WriteString(p.PType)
	if p.V0 != "" {
		sb.WriteString(", ")
		sb.WriteString(p.V0)
	}
	if p.V1 != "" {
		sb.WriteString(", ")
		sb.WriteString(p.V1)
	}
	if p.V2 != "" {
		sb.WriteString(", ")
		sb.WriteString(p.V2)
	}
	if p.V3 != "" {
		sb.WriteString(", ")
		sb.WriteString(p.V3)
	}
	if p.V4 != "" {
		sb.WriteString(", ")
		sb.WriteString(p.V4)
	}
	if p.V5 != "" {
		sb.WriteString(", ")
		sb.WriteString(p.V5)
	}
	return sb.String()
}

func stringToPolicy(ptype string, rule []string) *Policy {
	p := &Policy{PType: ptype}
	if len(rule) > 0 {
		p.V0 = rule[0]
	}
	if len(rule) > 1 {
		p.V1 = rule[1]
	}
	if len(rule) > 2 {
		p.V2 = rule[2]
	}
	if len(rule) > 3 {
		p.V3 = rule[3]
	}
	if len(rule) > 4 {
		p.V4 = rule[4]
	}
	if len(rule) > 5 {
		p.V5 = rule[5]
	}
	return p
}

// Ensure Adapter implements persist.Adapter
var _ persist.Adapter = &Adapter{}
