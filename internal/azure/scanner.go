package azure

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/kjourdan1/lzctl/internal/audit"
)

// Scanner inventories Azure tenant resources via az CLI.
type Scanner struct {
	cli            CLI
	scope          string
	maxConcurrency int
}

// NewScanner creates an Azure scanner.
func NewScanner(cli CLI, scope string) *Scanner {
	if cli == nil {
		cli = NewAzCLI()
	}
	return &Scanner{cli: cli, scope: strings.TrimSpace(scope), maxConcurrency: 5}
}

// Scan collects tenant snapshot data used by compliance rules.
func (s *Scanner) Scan() (*audit.TenantSnapshot, []string, error) {
	snapshot := &audit.TenantSnapshot{}
	warnings := make([]string, 0)

	mgs, err := s.scanManagementGroups()
	if err != nil {
		warnings = append(warnings, "management groups: "+err.Error())
	} else {
		snapshot.ManagementGroups = mgs
	}

	subs, err := s.scanSubscriptions()
	if err != nil {
		return nil, warnings, err
	}
	snapshot.Subscriptions = subs

	policies, err := s.scanPolicyAssignments()
	if err != nil {
		warnings = append(warnings, "policy assignments: "+err.Error())
	} else {
		snapshot.PolicyAssignments = policies
	}

	rbac, err := s.scanRoleAssignments()
	if err != nil {
		warnings = append(warnings, "role assignments: "+err.Error())
	} else {
		snapshot.RoleAssignments = rbac
	}

	vnets, peerings, netWarnings := s.scanNetworking(subs)
	snapshot.VirtualNetworks = vnets
	snapshot.Peerings = peerings
	warnings = append(warnings, netWarnings...)

	diagnostics, err := s.scanDiagnosticSettings(subs)
	if err != nil {
		warnings = append(warnings, "diagnostic settings: "+err.Error())
	} else {
		snapshot.DiagnosticSettings = diagnostics
	}

	defender, err := s.scanDefenderPlans(subs)
	if err != nil {
		warnings = append(warnings, "defender plans: "+err.Error())
	} else {
		snapshot.DefenderPlans = defender
	}

	sort.Strings(warnings)
	return snapshot, warnings, nil
}

func (s *Scanner) scanManagementGroups() ([]audit.ManagementGroup, error) {
	raw, err := s.cli.RunJSON("account", "management-group", "list")
	if err != nil {
		return nil, err
	}
	items := asSlice(raw)
	out := make([]audit.ManagementGroup, 0, len(items))
	for _, item := range items {
		m := asMap(item)
		out = append(out, audit.ManagementGroup{
			ID:          asString(m["id"]),
			Name:        asString(m["name"]),
			DisplayName: asString(m["displayName"]),
			ParentID:    asString(m["parentId"]),
		})
	}
	return out, nil
}

func (s *Scanner) scanSubscriptions() ([]audit.Subscription, error) {
	raw, err := s.cli.RunJSON("account", "list")
	if err != nil {
		return nil, err
	}
	items := asSlice(raw)
	out := make([]audit.Subscription, 0, len(items))
	for _, item := range items {
		m := asMap(item)
		sub := audit.Subscription{
			ID:              asString(m["id"]),
			DisplayName:     asString(m["name"]),
			State:           asString(m["state"]),
			ManagementGroup: asString(m["managementGroupId"]),
		}
		if sub.ID != "" {
			out = append(out, sub)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no subscriptions returned by az account list")
	}
	return out, nil
}

func (s *Scanner) scanPolicyAssignments() ([]audit.PolicyAssignment, error) {
	args := []string{"policy", "assignment", "list", "--all"}
	if s.scope != "" {
		args = append(args, "--scope", s.scope)
	}
	raw, err := s.cli.RunJSON(args...)
	if err != nil {
		return nil, err
	}
	items := asSlice(raw)
	out := make([]audit.PolicyAssignment, 0, len(items))
	for _, item := range items {
		m := asMap(item)
		out = append(out, audit.PolicyAssignment{
			ID:                 asString(m["id"]),
			Name:               asString(m["name"]),
			Scope:              asString(m["scope"]),
			PolicyDefinitionID: asString(m["policyDefinitionId"]),
		})
	}
	return out, nil
}

func (s *Scanner) scanRoleAssignments() ([]audit.RoleAssignment, error) {
	raw, err := s.cli.RunJSON("role", "assignment", "list", "--all")
	if err != nil {
		return nil, err
	}
	items := asSlice(raw)
	out := make([]audit.RoleAssignment, 0, len(items))
	for _, item := range items {
		m := asMap(item)
		out = append(out, audit.RoleAssignment{
			ID:                   asString(m["id"]),
			PrincipalID:          asString(m["principalId"]),
			PrincipalType:        asString(m["principalType"]),
			RoleDefinitionName:   asString(m["roleDefinitionName"]),
			Scope:                asString(m["scope"]),
			HasFederatedIdentity: asBool(m["hasFederatedCredential"]),
		})
	}
	return out, nil
}

func (s *Scanner) scanNetworking(subs []audit.Subscription) ([]audit.VirtualNetwork, []audit.VNetPeering, []string) {
	vnets := make([]audit.VirtualNetwork, 0)
	peerings := make([]audit.VNetPeering, 0)
	warnings := make([]string, 0)

	type result struct {
		vnets    []audit.VirtualNetwork
		peerings []audit.VNetPeering
		warn     string
	}
	resCh := make(chan result, len(subs))
	sem := make(chan struct{}, s.maxConcurrency)
	var wg sync.WaitGroup

	for _, sub := range subs {
		subscriptionID := sub.ID
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			raw, err := s.cli.RunJSON("network", "vnet", "list", "--subscription", subscriptionID)
			if err != nil {
				resCh <- result{warn: fmt.Sprintf("network %s: %v", subscriptionID, err)}
				return
			}
			items := asSlice(raw)
			localVNets := make([]audit.VirtualNetwork, 0, len(items))
			localPeerings := make([]audit.VNetPeering, 0)
			for _, item := range items {
				m := asMap(item)
				v := audit.VirtualNetwork{
					ID:             asString(m["id"]),
					Name:           asString(m["name"]),
					SubscriptionID: subscriptionID,
					AddressSpaces:  asStringSlicePath(m, "addressSpace", "addressPrefixes"),
				}
				localVNets = append(localVNets, v)

				for _, peering := range asSlice(m["virtualNetworkPeerings"]) {
					pm := asMap(peering)
					localPeerings = append(localPeerings, audit.VNetPeering{
						Name:                asString(pm["name"]),
						VirtualNetworkID:    v.ID,
						RemoteNetworkID:     asStringPath(pm, "remoteVirtualNetwork", "id"),
						PeeringState:        asString(pm["peeringState"]),
						AllowGatewayTransit: asBool(pm["allowGatewayTransit"]),
					})
				}
			}
			resCh <- result{vnets: localVNets, peerings: localPeerings}
		}()
	}

	wg.Wait()
	close(resCh)

	for r := range resCh {
		if r.warn != "" {
			warnings = append(warnings, r.warn)
			continue
		}
		vnets = append(vnets, r.vnets...)
		peerings = append(peerings, r.peerings...)
	}

	return vnets, peerings, warnings
}

func (s *Scanner) scanDiagnosticSettings(subs []audit.Subscription) ([]audit.DiagnosticSetting, error) { //nolint:unparam // error return reserved for future use
	out := make([]audit.DiagnosticSetting, 0, len(subs))
	for _, sub := range subs {
		resourceID := "/subscriptions/" + sub.ID
		raw, err := s.cli.RunJSON("monitor", "diagnostic-settings", "list", "--resource", resourceID)
		if err != nil {
			continue
		}
		for _, item := range asSlice(raw) {
			m := asMap(item)
			out = append(out, audit.DiagnosticSetting{
				ID:             asString(m["id"]),
				Name:           asString(m["name"]),
				Scope:          resourceID,
				WorkspaceID:    asString(m["workspaceId"]),
				SubscriptionID: sub.ID,
			})
		}
	}
	return out, nil
}

func (s *Scanner) scanDefenderPlans(subs []audit.Subscription) ([]audit.DefenderPlan, error) { //nolint:unparam // error return reserved for future use
	out := make([]audit.DefenderPlan, 0)
	for _, sub := range subs {
		raw, err := s.cli.RunJSON("security", "pricing", "list", "--subscription", sub.ID)
		if err != nil {
			continue
		}
		for _, item := range asSlice(raw) {
			m := asMap(item)
			out = append(out, audit.DefenderPlan{
				SubscriptionID: sub.ID,
				Name:           asString(m["name"]),
				PricingTier:    asString(m["pricingTier"]),
			})
		}
	}
	return out, nil
}

func asSlice(v any) []any {
	if v == nil {
		return nil
	}
	if arr, ok := v.([]any); ok {
		return arr
	}
	return nil
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asBool(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func asStringPath(m map[string]any, path ...string) string {
	var curr any = m
	for _, key := range path {
		mm, ok := curr.(map[string]any)
		if !ok {
			return ""
		}
		curr = mm[key]
	}
	return asString(curr)
}

func asStringSlicePath(m map[string]any, first string, second string) []string {
	nested, ok := m[first].(map[string]any)
	if !ok {
		return nil
	}
	items := asSlice(nested[second])
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s := asString(item); s != "" {
			out = append(out, s)
		}
	}
	return out
}
