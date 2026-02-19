package audit

import (
	"net"
	"strings"
)

type staticRule struct {
	id         string
	discipline string
	eval       func(*TenantSnapshot) []AuditFinding
}

func (r staticRule) ID() string         { return r.id }
func (r staticRule) Discipline() string { return r.discipline }
func (r staticRule) Evaluate(snapshot *TenantSnapshot) []AuditFinding {
	return r.eval(snapshot)
}

func defaultRules() []ComplianceRule {
	return []ComplianceRule{
		ruleGOV001(), ruleGOV002(), ruleGOV003(), ruleGOV004(),
		ruleIDT001(), ruleIDT002(),
		ruleMGT001(), ruleMGT002(), ruleMGT003(),
		ruleNET001(), ruleNET002(), ruleNET003(),
		ruleSEC001(), ruleSEC002(),
	}
}

func ruleGOV001() ComplianceRule {
	return staticRule{id: "GOV-001", discipline: "governance", eval: func(s *TenantSnapshot) []AuditFinding {
		hasPlatform, hasLandingZones := false, false
		for _, mg := range s.ManagementGroups {
			name := strings.ToLower(mg.DisplayName + " " + mg.Name)
			if strings.Contains(name, "platform") {
				hasPlatform = true
			}
			if strings.Contains(name, "landing") {
				hasLandingZones = true
			}
		}
		if hasPlatform && hasLandingZones {
			return nil
		}
		return []AuditFinding{{
			ID:            "GOV-001",
			Discipline:    "governance",
			Severity:      "high",
			Title:         "CAF management group hierarchy is incomplete",
			CurrentState:  "Required management group branches were not found",
			ExpectedState: "Platform and Landing Zones branches should exist",
			Remediation:   "Create CAF-aligned management groups",
			AutoFixable:   true,
		}}
	}}
}

func ruleGOV002() ComplianceRule {
	return staticRule{id: "GOV-002", discipline: "governance", eval: func(s *TenantSnapshot) []AuditFinding {
		findings := make([]AuditFinding, 0)
		for _, sub := range s.Subscriptions {
			if strings.TrimSpace(sub.ManagementGroup) == "" {
				findings = append(findings, AuditFinding{ID: "GOV-002", Discipline: "governance", Severity: "medium", Title: "Subscription has no management group", CurrentState: sub.DisplayName, ExpectedState: "Each subscription should be placed under a CAF management group", Remediation: "Move subscription to the right management group", AutoFixable: false})
			}
		}
		return findings
	}}
}

func ruleGOV003() ComplianceRule {
	return staticRule{id: "GOV-003", discipline: "governance", eval: func(s *TenantSnapshot) []AuditFinding {
		for _, p := range s.PolicyAssignments {
			id := strings.ToLower(p.PolicyDefinitionID + " " + p.Name)
			if strings.Contains(id, "caf") || strings.Contains(id, "mdfc") || strings.Contains(id, "deploy") {
				return nil
			}
		}
		return []AuditFinding{{ID: "GOV-003", Discipline: "governance", Severity: "high", Title: "CAF baseline policies not found", CurrentState: "No CAF-like policy assignment detected", ExpectedState: "CAF baseline initiatives assigned", Remediation: "Assign CAF baseline policy initiatives", AutoFixable: true}}
	}}
}

func ruleGOV004() ComplianceRule {
	return staticRule{id: "GOV-004", discipline: "governance", eval: func(s *TenantSnapshot) []AuditFinding {
		findings := make([]AuditFinding, 0)
		for _, sub := range s.Subscriptions {
			mg := strings.ToLower(sub.ManagementGroup)
			if strings.Contains(mg, "tenant root") || strings.HasSuffix(mg, "/") || mg == "root" {
				findings = append(findings, AuditFinding{ID: "GOV-004", Discipline: "governance", Severity: "critical", Title: "Subscription at root scope", CurrentState: sub.DisplayName + " assigned at root", ExpectedState: "Subscriptions should be under dedicated child management groups", Remediation: "Move subscription under CAF child group", AutoFixable: false})
			}
		}
		return findings
	}}
}

func ruleIDT001() ComplianceRule {
	return staticRule{id: "IDT-001", discipline: "identity", eval: func(s *TenantSnapshot) []AuditFinding {
		for _, ra := range s.RoleAssignments {
			if strings.EqualFold(ra.RoleDefinitionName, "Owner") && strings.Contains(strings.ToLower(ra.Scope), "/providers/microsoft.management/managementgroups") {
				return []AuditFinding{{ID: "IDT-001", Discipline: "identity", Severity: "critical", Title: "Persistent Owner role on management group", CurrentState: "Owner role assignment detected at management group scope", ExpectedState: "Use least-privilege role and PIM for elevated access", Remediation: "Remove direct Owner assignments on management groups", AutoFixable: false}}
			}
		}
		return nil
	}}
}

func ruleIDT002() ComplianceRule {
	return staticRule{id: "IDT-002", discipline: "identity", eval: func(s *TenantSnapshot) []AuditFinding {
		for _, ra := range s.RoleAssignments {
			if strings.EqualFold(ra.PrincipalType, "ServicePrincipal") && !ra.HasFederatedIdentity {
				return []AuditFinding{{ID: "IDT-002", Discipline: "identity", Severity: "medium", Title: "Service principal without federated identity", CurrentState: "At least one service principal lacks federated identity", ExpectedState: "Use workload identity federation for CI/CD principals", Remediation: "Add federated credentials and remove static secrets", AutoFixable: true}}
			}
		}
		return nil
	}}
}

func ruleMGT001() ComplianceRule {
	return staticRule{id: "MGT-001", discipline: "management", eval: func(s *TenantSnapshot) []AuditFinding {
		for _, d := range s.DiagnosticSettings {
			if strings.TrimSpace(d.WorkspaceID) != "" {
				return nil
			}
		}
		return []AuditFinding{{ID: "MGT-001", Discipline: "management", Severity: "high", Title: "Log Analytics workspace linkage not detected", CurrentState: "Diagnostic settings are missing workspace references", ExpectedState: "Subscriptions should stream diagnostics to Log Analytics", Remediation: "Create Log Analytics workspace and connect diagnostics", AutoFixable: true}}
	}}
}

func ruleMGT002() ComplianceRule {
	return staticRule{id: "MGT-002", discipline: "management", eval: func(s *TenantSnapshot) []AuditFinding {
		covered := map[string]bool{}
		for _, d := range s.DiagnosticSettings {
			covered[d.SubscriptionID] = true
		}
		for _, sub := range s.Subscriptions {
			if !covered[sub.ID] {
				return []AuditFinding{{ID: "MGT-002", Discipline: "management", Severity: "medium", Title: "Diagnostic settings missing for subscriptions", CurrentState: "At least one subscription has no diagnostic setting", ExpectedState: "Each subscription should have diagnostic settings configured", Remediation: "Enable subscription-level diagnostics", AutoFixable: true}}
			}
		}
		return nil
	}}
}

func ruleMGT003() ComplianceRule {
	return staticRule{id: "MGT-003", discipline: "management", eval: func(s *TenantSnapshot) []AuditFinding {
		for _, plan := range s.DefenderPlans {
			if strings.EqualFold(plan.PricingTier, "standard") {
				return nil
			}
		}
		return []AuditFinding{{ID: "MGT-003", Discipline: "management", Severity: "high", Title: "Defender for Cloud standard plans not detected", CurrentState: "No standard Defender pricing tiers detected", ExpectedState: "Defender standard plans enabled for critical resource types", Remediation: "Enable Defender standard plans", AutoFixable: true}}
	}}
}

func ruleNET001() ComplianceRule {
	return staticRule{id: "NET-001", discipline: "connectivity", eval: func(s *TenantSnapshot) []AuditFinding {
		for _, v := range s.VirtualNetworks {
			if strings.Contains(strings.ToLower(v.Name), "hub") {
				return nil
			}
		}
		return []AuditFinding{{ID: "NET-001", Discipline: "connectivity", Severity: "high", Title: "Hub virtual network not found", CurrentState: "No hub-like VNet name detected", ExpectedState: "A hub VNet should exist for shared services", Remediation: "Create or identify hub network topology", AutoFixable: true}}
	}}
}

func ruleNET002() ComplianceRule {
	return staticRule{id: "NET-002", discipline: "connectivity", eval: func(s *TenantSnapshot) []AuditFinding {
		for _, p := range s.Peerings {
			if strings.EqualFold(p.PeeringState, "Connected") {
				return nil
			}
		}
		return []AuditFinding{{ID: "NET-002", Discipline: "connectivity", Severity: "medium", Title: "No connected VNet peering detected", CurrentState: "Peering state not connected or missing", ExpectedState: "Hub-spoke peerings should be connected", Remediation: "Create and validate VNet peerings", AutoFixable: true}}
	}}
}

func ruleNET003() ComplianceRule {
	return staticRule{id: "NET-003", discipline: "connectivity", eval: func(s *TenantSnapshot) []AuditFinding {
		nets := make([]*net.IPNet, 0)
		for _, v := range s.VirtualNetworks {
			for _, cidr := range v.AddressSpaces {
				_, ipn, err := net.ParseCIDR(cidr)
				if err == nil {
					nets = append(nets, ipn)
				}
			}
		}
		for i := 0; i < len(nets); i++ {
			for j := i + 1; j < len(nets); j++ {
				if nets[i].Contains(nets[j].IP) || nets[j].Contains(nets[i].IP) {
					return []AuditFinding{{ID: "NET-003", Discipline: "connectivity", Severity: "critical", Title: "Overlapping VNet address spaces detected", CurrentState: "At least two VNets overlap", ExpectedState: "VNet address spaces should be unique", Remediation: "Re-address conflicting VNets", AutoFixable: false}}
				}
			}
		}
		return nil
	}}
}

func ruleSEC001() ComplianceRule {
	return staticRule{id: "SEC-001", discipline: "security", eval: func(s *TenantSnapshot) []AuditFinding {
		for _, p := range s.PolicyAssignments {
			id := strings.ToLower(p.PolicyDefinitionID + " " + p.Name)
			if strings.Contains(id, "tls") || strings.Contains(id, "storage") {
				return nil
			}
		}
		return []AuditFinding{{ID: "SEC-001", Discipline: "security", Severity: "medium", Title: "Storage TLS governance policy not detected", CurrentState: "No storage/TLS baseline policy assignment found", ExpectedState: "Policies enforcing secure transport should be assigned", Remediation: "Assign storage TLS baseline policy", AutoFixable: true}}
	}}
}

func ruleSEC002() ComplianceRule {
	return staticRule{id: "SEC-002", discipline: "security", eval: func(s *TenantSnapshot) []AuditFinding {
		for _, p := range s.PolicyAssignments {
			id := strings.ToLower(p.PolicyDefinitionID + " " + p.Name)
			if strings.Contains(id, "keyvault") || strings.Contains(id, "soft delete") {
				return nil
			}
		}
		return []AuditFinding{{ID: "SEC-002", Discipline: "security", Severity: "medium", Title: "Key Vault soft delete policy not detected", CurrentState: "No key vault soft delete policy assignment found", ExpectedState: "Key Vaults should enforce soft delete and purge protection", Remediation: "Assign key vault security baseline policies", AutoFixable: true}}
	}}
}
