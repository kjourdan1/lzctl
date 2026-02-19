package audit

import "encoding/json"

// RenderJSON renders the audit report as indented JSON bytes.
func RenderJSON(report *AuditReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}
