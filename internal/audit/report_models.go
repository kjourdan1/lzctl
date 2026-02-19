package audit

import "time"

type TenantSnapshot struct {
	TenantID           string
	ManagementGroups   []ManagementGroup
	Subscriptions      []Subscription
	PolicyAssignments  []PolicyAssignment
	RoleAssignments    []RoleAssignment
	VirtualNetworks    []VirtualNetwork
	Peerings           []VNetPeering
	DiagnosticSettings []DiagnosticSetting
	DefenderPlans      []DefenderPlan
	ScannedAt          time.Time
}

type ManagementGroup struct {
	ID          string
	Name        string
	DisplayName string
	ParentID    string
}

type Subscription struct {
	ID              string
	DisplayName     string
	State           string
	ManagementGroup string
}

type PolicyAssignment struct {
	ID                 string
	Name               string
	Scope              string
	PolicyDefinitionID string
}

type RoleAssignment struct {
	ID                   string
	PrincipalID          string
	PrincipalType        string
	RoleDefinitionName   string
	Scope                string
	HasFederatedIdentity bool
}

type VirtualNetwork struct {
	ID             string
	Name           string
	SubscriptionID string
	AddressSpaces  []string
}

type VNetPeering struct {
	Name                string
	VirtualNetworkID    string
	RemoteNetworkID     string
	PeeringState        string
	AllowGatewayTransit bool
}

type DiagnosticSetting struct {
	ID             string
	Name           string
	Scope          string
	WorkspaceID    string
	SubscriptionID string
}

type DefenderPlan struct {
	SubscriptionID string
	Name           string
	PricingTier    string
}

type AuditFinding struct {
	ID            string        `json:"id"`
	Discipline    string        `json:"discipline"`
	Severity      string        `json:"severity"`
	Title         string        `json:"title"`
	CurrentState  string        `json:"currentState"`
	ExpectedState string        `json:"expectedState"`
	Remediation   string        `json:"remediation"`
	AutoFixable   bool          `json:"autoFixable"`
	Resources     []ResourceRef `json:"resources,omitempty"`
}

type ResourceRef struct {
	ResourceID   string `json:"resourceId"`
	ResourceType string `json:"resourceType"`
	Name         string `json:"name"`
}

type AuditScore struct {
	Overall      int `json:"overall"`
	Governance   int `json:"governance"`
	Identity     int `json:"identity"`
	Management   int `json:"management"`
	Connectivity int `json:"connectivity"`
	Security     int `json:"security"`
}

type AuditSummary struct {
	TotalFindings int `json:"totalFindings"`
	Critical      int `json:"critical"`
	High          int `json:"high"`
	Medium        int `json:"medium"`
	Low           int `json:"low"`
	AutoFixable   int `json:"autoFixable"`
}

type AuditReport struct {
	TenantID  string         `json:"tenantId"`
	ScannedAt time.Time      `json:"scannedAt"`
	Score     AuditScore     `json:"score"`
	Findings  []AuditFinding `json:"findings"`
	Summary   AuditSummary   `json:"summary"`
}
