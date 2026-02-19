package upgrade

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ModulePin represents a module source + version pin found in a .tf file.
type ModulePin struct {
	Ref      ModuleRef
	Version  string
	FilePath string
	Line     int
}

var (
	// Matches: source = "registry.terraform.io/Azure/avm-xxx/azurerm"
	// or source = "Azure/avm-xxx/azurerm"
	sourceRE = regexp.MustCompile(`source\s*=\s*"(?:registry\.terraform\.io/)?([^/]+)/([^/]+)/([^"]+)"`)
	// Matches: version = "~> 0.4.0" or version = ">= 1.0.0" or version = "1.2.3"
	versionRE = regexp.MustCompile(`version\s*=\s*"([^"]*)"`)
)

// ScanDirectory walks a directory tree and extracts all module version pins
// from .tf files.
func ScanDirectory(root string) ([]ModulePin, error) {
	var pins []ModulePin

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".terraform" || base == ".git" || base == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".tf") {
			return nil
		}

		filePins, scanErr := scanFile(path)
		if scanErr != nil {
			return nil // skip files that can't be read
		}
		pins = append(pins, filePins...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("scanning directory %s: %w", root, err)
	}
	return pins, nil
}

func scanFile(path string) ([]ModulePin, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var pins []ModulePin
	scanner := bufio.NewScanner(f)

	var currentSource *ModuleRef
	var sourceLine int
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Look for module source.
		if matches := sourceRE.FindStringSubmatch(trimmed); matches != nil {
			currentSource = &ModuleRef{
				Namespace: matches[1],
				Name:      matches[2],
				Provider:  matches[3],
			}
			sourceLine = lineNum
			continue
		}

		// Look for version pin after a source.
		if currentSource != nil {
			if matches := versionRE.FindStringSubmatch(trimmed); matches != nil {
				version := extractVersion(matches[1])
				pins = append(pins, ModulePin{
					Ref:      *currentSource,
					Version:  version,
					FilePath: path,
					Line:     sourceLine,
				})
				currentSource = nil
				continue
			}
		}

		// Reset if we hit closing brace (end of module block).
		if trimmed == "}" {
			currentSource = nil
		}
	}

	return pins, scanner.Err()
}

// extractVersion strips constraint operators from version strings.
// "~> 0.4.0" → "0.4.0", ">= 1.0.0" → "1.0.0"
func extractVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "~>")
	v = strings.TrimPrefix(v, ">=")
	v = strings.TrimPrefix(v, "<=")
	v = strings.TrimPrefix(v, ">")
	v = strings.TrimPrefix(v, "<")
	v = strings.TrimPrefix(v, "=")
	return strings.TrimSpace(v)
}

// UpdateVersionInFile updates a module version pin in a .tf file.
// It finds the source line and updates the following version line.
func UpdateVersionInFile(pin ModulePin, newVersion string) error {
	data, err := os.ReadFile(pin.FilePath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", pin.FilePath, err)
	}

	lines := strings.Split(string(data), "\n")
	updated := false

	// Find the source line and update the next version line.
	foundSource := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if sourceRE.MatchString(trimmed) {
			matches := sourceRE.FindStringSubmatch(trimmed)
			if matches != nil && matches[1] == pin.Ref.Namespace && matches[2] == pin.Ref.Name {
				foundSource = true
				continue
			}
		}

		if foundSource && versionRE.MatchString(trimmed) {
			// Replace version value preserving constraint operators and indentation.
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			oldVersionMatch := versionRE.FindStringSubmatch(trimmed)
			if oldVersionMatch != nil {
				oldConstraint := oldVersionMatch[1]
				// Preserve constraint operator.
				prefix := ""
				if strings.HasPrefix(strings.TrimSpace(oldConstraint), "~>") {
					prefix = "~> "
				} else if strings.HasPrefix(strings.TrimSpace(oldConstraint), ">=") {
					prefix = ">= "
				}
				lines[i] = fmt.Sprintf(`%sversion = "%s%s"`, indent, prefix, newVersion)
				updated = true
				break
			}
		}

		if foundSource && trimmed == "}" {
			foundSource = false
		}
	}

	if !updated {
		return fmt.Errorf("version line not found for module %s in %s", pin.Ref, pin.FilePath)
	}

	return os.WriteFile(pin.FilePath, []byte(strings.Join(lines, "\n")), 0o600)
}
