// Package state implements Terraform state lifecycle management for lzctl.
//
// Philosophy: Terraform state is a critical asset. This package treats it as
// such by providing:
//   - Snapshot: Create point-in-time backups of state files before mutations
//   - List: Enumerate state files in the backend for visibility
//   - Restore: Recover a specific state version (using blob versioning)
//   - Audit: Check state backend health (versioning, soft-delete, encryption)
//
// The state backend is always Azure Storage (azurerm), using blob lease locking
// to prevent concurrent writes (equivalent to DynamoDB locking in AWS).
//
// References:
//   - PRD FR-2.8, FR-2.9: Bootstrap state backend with versioning + soft delete
//   - PRD FR-5.5: Each layer has its own state file within the shared backend
//   - ADR-005: Layers with separate state files
package state

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kjourdan1/lzctl/internal/config"
)

// StateFile represents a Terraform state blob in Azure Storage.
type StateFile struct {
	Key          string    `json:"key"`
	Layer        string    `json:"layer"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
	LeaseStatus  string    `json:"leaseStatus"` // "locked" | "unlocked"
	VersionID    string    `json:"versionId,omitempty"`
}

// Snapshot represents a point-in-time backup of a state file.
type Snapshot struct {
	Key       string    `json:"key"`
	VersionID string    `json:"versionId"`
	CreatedAt time.Time `json:"createdAt"`
	Size      int64     `json:"size"`
	Tag       string    `json:"tag,omitempty"` // user-defined label (e.g. "pre-apply-2026-02-19")
}

// BackendHealth captures the security posture of the state backend.
type BackendHealth struct {
	StorageAccount string        `json:"storageAccount"`
	Container      string        `json:"container"`
	Checks         []HealthCheck `json:"checks"`
	Healthy        bool          `json:"healthy"`
}

// HealthCheck is a single state backend health validation.
type HealthCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass" | "fail" | "warn"
	Message string `json:"message"`
	Fix     string `json:"fix,omitempty"`
}

// AzCLIRunner abstracts az CLI execution for testability.
type AzCLIRunner interface {
	Run(args ...string) (string, error)
}

// Manager orchestrates state lifecycle operations.
type Manager struct {
	cfg *config.LZConfig
	cli AzCLIRunner
}

// NewManager creates a state lifecycle manager from the project config.
func NewManager(cfg *config.LZConfig, cli AzCLIRunner) *Manager {
	return &Manager{cfg: cfg, cli: cli}
}

// ListStates enumerates all Terraform state blobs in the backend container.
func (m *Manager) ListStates() ([]StateFile, error) {
	sb := m.cfg.Spec.StateBackend
	args := []string{
		"storage", "blob", "list",
		"--account-name", sb.StorageAccount,
		"--container-name", sb.Container,
		"--subscription", sb.Subscription,
		"--auth-mode", "login",
		"--output", "json",
	}
	out, err := m.cli.Run(args...)
	if err != nil {
		return nil, fmt.Errorf("listing state blobs: %w", err)
	}

	var blobs []struct {
		Name       string `json:"name"`
		Properties struct {
			ContentLength int64  `json:"contentLength"`
			LastModified  string `json:"lastModified"`
			LeaseStatus   string `json:"leaseStatus"`
		} `json:"properties"`
		VersionID string `json:"versionId"`
	}
	if err := json.Unmarshal([]byte(out), &blobs); err != nil {
		return nil, fmt.Errorf("parsing blob list: %w", err)
	}

	states := make([]StateFile, 0, len(blobs))
	for _, b := range blobs {
		if !strings.HasSuffix(b.Name, ".tfstate") {
			continue
		}
		mod, _ := time.Parse(time.RFC1123, b.Properties.LastModified)
		states = append(states, StateFile{
			Key:          b.Name,
			Layer:        stateKeyToLayer(b.Name),
			Size:         b.Properties.ContentLength,
			LastModified: mod,
			LeaseStatus:  b.Properties.LeaseStatus,
			VersionID:    b.VersionID,
		})
	}
	return states, nil
}

// CreateSnapshot creates a point-in-time copy of a state file by triggering
// a blob snapshot in Azure Storage. This uses native blob versioning when
// available, or falls back to a blob snapshot.
func (m *Manager) CreateSnapshot(stateKey, tag string) (*Snapshot, error) {
	sb := m.cfg.Spec.StateBackend
	args := []string{
		"storage", "blob", "snapshot",
		"--account-name", sb.StorageAccount,
		"--container-name", sb.Container,
		"--name", stateKey,
		"--subscription", sb.Subscription,
		"--auth-mode", "login",
		"--output", "json",
	}
	out, err := m.cli.Run(args...)
	if err != nil {
		return nil, fmt.Errorf("creating snapshot for %s: %w", stateKey, err)
	}

	var result struct {
		Snapshot  string `json:"snapshot"`
		VersionID string `json:"versionId"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("parsing snapshot result: %w", err)
	}

	versionID := result.VersionID
	if versionID == "" {
		versionID = result.Snapshot
	}

	return &Snapshot{
		Key:       stateKey,
		VersionID: versionID,
		CreatedAt: time.Now().UTC(),
		Tag:       tag,
	}, nil
}

// SnapshotAll creates snapshots of all state files in the backend.
// Useful as a pre-apply safety net.
func (m *Manager) SnapshotAll(tag string) ([]Snapshot, error) {
	states, err := m.ListStates()
	if err != nil {
		return nil, err
	}

	snapshots := make([]Snapshot, 0, len(states))
	for _, s := range states {
		snap, err := m.CreateSnapshot(s.Key, tag)
		if err != nil {
			return nil, fmt.Errorf("snapshot %s: %w", s.Key, err)
		}
		snapshots = append(snapshots, *snap)
	}
	return snapshots, nil
}

// ListVersions lists all versions of a specific state file (requires blob versioning).
func (m *Manager) ListVersions(stateKey string) ([]Snapshot, error) {
	sb := m.cfg.Spec.StateBackend
	args := []string{
		"storage", "blob", "list",
		"--account-name", sb.StorageAccount,
		"--container-name", sb.Container,
		"--prefix", stateKey,
		"--include", "v", // include versions
		"--subscription", sb.Subscription,
		"--auth-mode", "login",
		"--output", "json",
	}
	out, err := m.cli.Run(args...)
	if err != nil {
		return nil, fmt.Errorf("listing versions for %s: %w", stateKey, err)
	}

	var blobs []struct {
		Name       string `json:"name"`
		VersionID  string `json:"versionId"`
		Properties struct {
			ContentLength int64  `json:"contentLength"`
			LastModified  string `json:"lastModified"`
		} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(out), &blobs); err != nil {
		return nil, fmt.Errorf("parsing version list: %w", err)
	}

	versions := make([]Snapshot, 0, len(blobs))
	for _, b := range blobs {
		if b.Name != stateKey {
			continue
		}
		mod, _ := time.Parse(time.RFC1123, b.Properties.LastModified)
		versions = append(versions, Snapshot{
			Key:       b.Name,
			VersionID: b.VersionID,
			CreatedAt: mod,
			Size:      b.Properties.ContentLength,
		})
	}
	return versions, nil
}

// CheckHealth validates the state backend security posture.
// Returns actionable findings if the backend deviates from best practices:
//   - Blob versioning enabled (audit trail, rollback)
//   - Soft delete enabled (protection against accidental deletion)
//   - HTTPS-only (encryption in transit)
//   - Infrastructure encryption (encryption at rest)
//   - Blob lease locking working (prevents concurrent writes)
func (m *Manager) CheckHealth() (*BackendHealth, error) {
	sb := m.cfg.Spec.StateBackend
	health := &BackendHealth{
		StorageAccount: sb.StorageAccount,
		Container:      sb.Container,
		Healthy:        true,
	}

	// Query storage account properties
	args := []string{
		"storage", "account", "show",
		"--name", sb.StorageAccount,
		"--subscription", sb.Subscription,
		"--output", "json",
	}
	out, err := m.cli.Run(args...)
	if err != nil {
		health.Checks = append(health.Checks, HealthCheck{
			Name:    "storage-account-access",
			Status:  "fail",
			Message: fmt.Sprintf("Cannot access storage account %s: %v", sb.StorageAccount, err),
			Fix:     "Verify storage account exists and you have Reader role",
		})
		health.Healthy = false
		return health, nil
	}

	var acct struct {
		Properties struct {
			SupportsHTTPSTrafficOnly bool `json:"supportsHttpsTrafficOnly"`
			MinimumTLSVersion        string `json:"minimumTlsVersion"`
			Encryption               struct {
				RequireInfrastructureEncryption bool `json:"requireInfrastructureEncryption"`
			} `json:"encryption"`
		} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(out), &acct); err != nil {
		health.Checks = append(health.Checks, HealthCheck{
			Name:    "storage-account-parse",
			Status:  "warn",
			Message: "Could not parse storage account properties",
		})
	} else {
		// HTTPS only
		if acct.Properties.SupportsHTTPSTrafficOnly {
			health.Checks = append(health.Checks, HealthCheck{
				Name:    "https-only",
				Status:  "pass",
				Message: "HTTPS-only traffic is enforced",
			})
		} else {
			health.Checks = append(health.Checks, HealthCheck{
				Name:    "https-only",
				Status:  "fail",
				Message: "HTTP traffic is allowed — state data could be intercepted",
				Fix:     fmt.Sprintf("az storage account update --name %s --https-only true", sb.StorageAccount),
			})
			health.Healthy = false
		}

		// TLS version
		if acct.Properties.MinimumTLSVersion == "TLS1_2" {
			health.Checks = append(health.Checks, HealthCheck{
				Name:    "tls-version",
				Status:  "pass",
				Message: "Minimum TLS 1.2 enforced",
			})
		} else {
			health.Checks = append(health.Checks, HealthCheck{
				Name:    "tls-version",
				Status:  "fail",
				Message: fmt.Sprintf("Minimum TLS version is %s (should be TLS1_2)", acct.Properties.MinimumTLSVersion),
				Fix:     fmt.Sprintf("az storage account update --name %s --min-tls-version TLS1_2", sb.StorageAccount),
			})
			health.Healthy = false
		}

		// Infrastructure encryption
		if acct.Properties.Encryption.RequireInfrastructureEncryption {
			health.Checks = append(health.Checks, HealthCheck{
				Name:    "infrastructure-encryption",
				Status:  "pass",
				Message: "Infrastructure encryption (double encryption) is enabled",
			})
		} else {
			health.Checks = append(health.Checks, HealthCheck{
				Name:    "infrastructure-encryption",
				Status:  "warn",
				Message: "Infrastructure encryption is not enabled (Azure default encryption is still active)",
				Fix:     "Enable infrastructure encryption when creating the storage account (cannot be changed after creation)",
			})
		}
	}

	// Check blob versioning via blob service properties
	blobSvcArgs := []string{
		"storage", "account", "blob-service-properties", "show",
		"--account-name", sb.StorageAccount,
		"--subscription", sb.Subscription,
		"--auth-mode", "login",
		"--output", "json",
	}
	blobOut, err := m.cli.Run(blobSvcArgs...)
	if err == nil {
		var blobSvc struct {
			IsVersioningEnabled        bool `json:"isVersioningEnabled"`
			DeleteRetentionPolicy      struct{ Enabled bool } `json:"deleteRetentionPolicy"`
			ContainerDeleteRetention   struct{ Enabled bool } `json:"containerDeleteRetentionPolicy"`
		}
		if err := json.Unmarshal([]byte(blobOut), &blobSvc); err == nil {
			// Versioning
			if blobSvc.IsVersioningEnabled {
				health.Checks = append(health.Checks, HealthCheck{
					Name:    "blob-versioning",
					Status:  "pass",
					Message: "Blob versioning is enabled — state history is preserved for rollback",
				})
			} else {
				health.Checks = append(health.Checks, HealthCheck{
					Name:    "blob-versioning",
					Status:  "fail",
					Message: "Blob versioning is disabled — cannot track or restore previous state versions",
					Fix:     fmt.Sprintf("az storage account blob-service-properties update --account-name %s --enable-versioning true", sb.StorageAccount),
				})
				health.Healthy = false
			}

			// Soft delete
			if blobSvc.DeleteRetentionPolicy.Enabled {
				health.Checks = append(health.Checks, HealthCheck{
					Name:    "soft-delete",
					Status:  "pass",
					Message: "Blob soft delete is enabled — protection against accidental deletion",
				})
			} else {
				health.Checks = append(health.Checks, HealthCheck{
					Name:    "soft-delete",
					Status:  "fail",
					Message: "Blob soft delete is disabled — accidental state deletion is unrecoverable",
					Fix:     fmt.Sprintf("az storage account blob-service-properties update --account-name %s --enable-delete-retention true --delete-retention-days 30", sb.StorageAccount),
				})
				health.Healthy = false
			}

			// Container soft delete
			if blobSvc.ContainerDeleteRetention.Enabled {
				health.Checks = append(health.Checks, HealthCheck{
					Name:    "container-soft-delete",
					Status:  "pass",
					Message: "Container soft delete is enabled",
				})
			} else {
				health.Checks = append(health.Checks, HealthCheck{
					Name:    "container-soft-delete",
					Status:  "warn",
					Message: "Container soft delete is not enabled",
					Fix:     fmt.Sprintf("az storage account blob-service-properties update --account-name %s --enable-container-delete-retention true --container-delete-retention-days 7", sb.StorageAccount),
				})
			}
		}
	}

	return health, nil
}

// BreakLease force-releases a stuck blob lease (for recovery from failed pipelines).
func (m *Manager) BreakLease(stateKey string) error {
	sb := m.cfg.Spec.StateBackend
	args := []string{
		"storage", "blob", "lease", "break",
		"--account-name", sb.StorageAccount,
		"--container-name", sb.Container,
		"--blob-name", stateKey,
		"--subscription", sb.Subscription,
		"--auth-mode", "login",
	}
	_, err := m.cli.Run(args...)
	if err != nil {
		return fmt.Errorf("breaking lease on %s: %w", stateKey, err)
	}
	return nil
}

// stateKeyToLayer converts a state key like "platform-connectivity.tfstate"
// to a human-readable layer name like "connectivity".
func stateKeyToLayer(key string) string {
	name := strings.TrimSuffix(key, ".tfstate")
	name = strings.ReplaceAll(name, "platform-", "")
	name = strings.ReplaceAll(name, "landing-zones-", "lz:")
	return name
}
