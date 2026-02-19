package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Event struct {
	Timestamp     string            `json:"timestamp"`
	Operation     string            `json:"operation"`
	Tenant        string            `json:"tenant,omitempty"`
	Ring          string            `json:"ring,omitempty"`
	Args          []string          `json:"args"`
	Result        string            `json:"result"`
	ExitCode      int               `json:"exitCode"`
	DurationMs    int64             `json:"durationMs"`
	CorrelationID string            `json:"correlationId"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

func BuildEvent(args []string, result string, exitCode int, duration time.Duration) Event {
	op, tenant, ring, repoRoot := inferFromArgs(args)
	meta := map[string]string{}
	if repoRoot != "" {
		meta["repoRoot"] = repoRoot
	}
	if len(meta) == 0 {
		meta = nil
	}
	return Event{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Operation:     op,
		Tenant:        tenant,
		Ring:          ring,
		Args:          args,
		Result:        result,
		ExitCode:      exitCode,
		DurationMs:    duration.Milliseconds(),
		CorrelationID: fmt.Sprintf("%d", time.Now().UTC().UnixNano()),
		Metadata:      meta,
	}
}

func Write(event Event) error {
	if err := writeUserAudit(event); err != nil {
		return err
	}
	if repoRoot := event.MetadataValue("repoRoot"); repoRoot != "" && event.Tenant != "" {
		_ = writeTenantAudit(repoRoot, event)
	}
	return nil
}

func ReadUserAudit() ([]Event, error) {
	path, err := userAuditPath()
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var out []Event
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err == nil {
			out = append(out, event)
		}
	}
	return out, scanner.Err()
}

func (e Event) MetadataValue(key string) string {
	if e.Metadata == nil {
		return ""
	}
	return e.Metadata[key]
}

func writeUserAudit(event Event) error {
	path, err := userAuditPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	line, err := json.Marshal(event)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

func writeTenantAudit(repoRoot string, event Event) error {
	dir := filepath.Join(repoRoot, "tenants", event.Tenant, "logs")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	file := filepath.Join(dir, fmt.Sprintf("%s-%s.json", sanitize(event.Operation), time.Now().UTC().Format("20060102-150405")))
	data, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o600)
}

func userAuditPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".lzctl", "audit.log"), nil
}

func inferFromArgs(args []string) (operation, tenant, ring, repoRoot string) {
	operation = "root"
	if len(args) > 1 {
		for i := 1; i < len(args); i++ {
			if strings.HasPrefix(args[i], "-") {
				continue
			}
			operation = args[i]
			break
		}
	}
	for i := 0; i < len(args); i++ {
		if i+1 < len(args) {
			switch args[i] {
			case "--tenant":
				tenant = args[i+1]
			case "--ring":
				ring = args[i+1]
			case "--repo-root":
				repoRoot = args[i+1]
			}
		}
	}
	if repoRoot == "" {
		repoRoot = "."
	}
	return
}

func sanitize(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "operation"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-")
	return replacer.Replace(s)
}
