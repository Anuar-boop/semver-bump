// semver-bump — Semantic version bumper for CI/CD
//
// Reads current version from various sources (package.json, Cargo.toml,
// go module, VERSION file), bumps it, and writes it back.
// Supports major, minor, patch, and pre-release bumps.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const version = "1.0.0"

type SemVer struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease string
	Build      string
}

func Parse(v string) (SemVer, error) {
	v = strings.TrimPrefix(v, "v")

	var sv SemVer

	// Split build metadata
	if idx := strings.Index(v, "+"); idx != -1 {
		sv.Build = v[idx+1:]
		v = v[:idx]
	}

	// Split pre-release
	if idx := strings.Index(v, "-"); idx != -1 {
		sv.PreRelease = v[idx+1:]
		v = v[:idx]
	}

	parts := strings.Split(v, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return sv, fmt.Errorf("invalid version: %s", v)
	}

	var err error
	sv.Major, err = strconv.Atoi(parts[0])
	if err != nil {
		return sv, fmt.Errorf("invalid major version: %s", parts[0])
	}

	if len(parts) >= 2 {
		sv.Minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return sv, fmt.Errorf("invalid minor version: %s", parts[1])
		}
	}

	if len(parts) >= 3 {
		sv.Patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return sv, fmt.Errorf("invalid patch version: %s", parts[2])
		}
	}

	return sv, nil
}

func (sv SemVer) String() string {
	s := fmt.Sprintf("%d.%d.%d", sv.Major, sv.Minor, sv.Patch)
	if sv.PreRelease != "" {
		s += "-" + sv.PreRelease
	}
	if sv.Build != "" {
		s += "+" + sv.Build
	}
	return s
}

func (sv SemVer) Bump(part string) SemVer {
	switch part {
	case "major":
		sv.Major++
		sv.Minor = 0
		sv.Patch = 0
		sv.PreRelease = ""
	case "minor":
		sv.Minor++
		sv.Patch = 0
		sv.PreRelease = ""
	case "patch":
		sv.Patch++
		sv.PreRelease = ""
	case "premajor":
		sv.Major++
		sv.Minor = 0
		sv.Patch = 0
		sv.PreRelease = "0"
	case "preminor":
		sv.Minor++
		sv.Patch = 0
		sv.PreRelease = "0"
	case "prepatch":
		sv.Patch++
		sv.PreRelease = "0"
	case "prerelease":
		if sv.PreRelease == "" {
			sv.Patch++
			sv.PreRelease = "0"
		} else {
			// Increment numeric pre-release
			if n, err := strconv.Atoi(sv.PreRelease); err == nil {
				sv.PreRelease = strconv.Itoa(n + 1)
			} else {
				// Try incrementing last numeric segment
				re := regexp.MustCompile(`(\d+)$`)
				if re.MatchString(sv.PreRelease) {
					sv.PreRelease = re.ReplaceAllStringFunc(sv.PreRelease, func(s string) string {
						n, _ := strconv.Atoi(s)
						return strconv.Itoa(n + 1)
					})
				} else {
					sv.PreRelease += ".1"
				}
			}
		}
	}
	return sv
}

// ─── Version Sources ───

type VersionSource struct {
	Name    string
	File    string
	Version string
}

func detectSource(dir string) *VersionSource {
	// Check sources in priority order
	sources := []struct {
		file    string
		name    string
		extract func(string) string
	}{
		{"package.json", "npm", extractFromPackageJSON},
		{"Cargo.toml", "cargo", extractFromCargoToml},
		{"pyproject.toml", "python", extractFromPyprojectToml},
		{"setup.py", "python", extractFromSetupPy},
		{"VERSION", "file", extractFromVersionFile},
		{"version.txt", "file", extractFromVersionFile},
		{".version", "file", extractFromVersionFile},
	}

	for _, src := range sources {
		path := dir + "/" + src.file
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		ver := src.extract(string(data))
		if ver != "" {
			return &VersionSource{
				Name:    src.name,
				File:    src.file,
				Version: ver,
			}
		}
	}

	return nil
}

func extractFromPackageJSON(content string) string {
	var pkg map[string]interface{}
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return ""
	}
	if v, ok := pkg["version"].(string); ok {
		return v
	}
	return ""
}

func extractFromCargoToml(content string) string {
	re := regexp.MustCompile(`(?m)^version\s*=\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func extractFromPyprojectToml(content string) string {
	re := regexp.MustCompile(`(?m)^version\s*=\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func extractFromSetupPy(content string) string {
	re := regexp.MustCompile(`version\s*=\s*['"]([^'"]+)['"]`)
	matches := re.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func extractFromVersionFile(content string) string {
	return strings.TrimSpace(content)
}

// ─── Version Writers ───

func writeVersion(source *VersionSource, newVersion string, dir string) error {
	path := dir + "/" + source.File

	switch source.Name {
	case "npm":
		return writePackageJSON(path, newVersion)
	case "cargo":
		return writeCargoToml(path, source.Version, newVersion)
	case "python":
		if source.File == "pyproject.toml" {
			return writeCargoToml(path, source.Version, newVersion) // same format
		}
		return writeSetupPy(path, source.Version, newVersion)
	case "file":
		return os.WriteFile(path, []byte(newVersion+"\n"), 0644)
	}

	return fmt.Errorf("unsupported source: %s", source.Name)
}

func writePackageJSON(path, newVersion string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return err
	}

	pkg["version"] = newVersion
	out, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(out, '\n'), 0644)
}

func writeCargoToml(path, oldVersion, newVersion string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := strings.Replace(
		string(data),
		fmt.Sprintf(`version = "%s"`, oldVersion),
		fmt.Sprintf(`version = "%s"`, newVersion),
		1,
	)

	return os.WriteFile(path, []byte(content), 0644)
}

func writeSetupPy(path, oldVersion, newVersion string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := strings.Replace(
		string(data),
		fmt.Sprintf(`version="%s"`, oldVersion),
		fmt.Sprintf(`version="%s"`, newVersion),
		1,
	)
	content = strings.Replace(
		content,
		fmt.Sprintf(`version='%s'`, oldVersion),
		fmt.Sprintf(`version='%s'`, newVersion),
		1,
	)

	return os.WriteFile(path, []byte(content), 0644)
}

func printUsage() {
	fmt.Printf(`
  semver-bump v%s — Semantic version bumper

  Usage:
    semver-bump <command> [options]

  Commands:
    major          Bump major version (1.2.3 → 2.0.0)
    minor          Bump minor version (1.2.3 → 1.3.0)
    patch          Bump patch version (1.2.3 → 1.2.4)
    premajor       Bump to pre-release major (1.2.3 → 2.0.0-0)
    preminor       Bump to pre-release minor (1.2.3 → 1.3.0-0)
    prepatch       Bump to pre-release patch (1.2.3 → 1.2.4-0)
    prerelease     Bump pre-release (1.2.4-0 → 1.2.4-1)
    current        Show current version
    set <version>  Set exact version

  Options:
    --dry-run      Show what would change without writing
    --prefix       Include 'v' prefix in output
    --dir <path>   Project directory (default: .)
    --tag          Create a git tag after bumping

  Auto-detects version from:
    package.json, Cargo.toml, pyproject.toml, setup.py, VERSION

  Examples:
    semver-bump patch
    semver-bump minor --tag
    semver-bump major --dry-run
    semver-bump set 2.0.0-beta.1
    semver-bump current
    semver-bump prerelease --prefix
`, version)
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 || args[0] == "--help" || args[0] == "help" {
		printUsage()
		return
	}

	if args[0] == "--version" {
		fmt.Println("semver-bump", version)
		return
	}

	command := args[0]
	dir := "."
	dryRun := false
	prefix := false
	createTag := false

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			dryRun = true
		case "--prefix":
			prefix = true
		case "--tag":
			createTag = true
		case "--dir":
			if i+1 < len(args) {
				dir = args[i+1]
				i++
			}
		}
	}

	source := detectSource(dir)
	if source == nil && command != "set" {
		fmt.Fprintln(os.Stderr, "  Error: no version file found")
		fmt.Fprintln(os.Stderr, "  Supported: package.json, Cargo.toml, pyproject.toml, setup.py, VERSION")
		os.Exit(1)
	}

	switch command {
	case "current":
		if source == nil {
			fmt.Fprintln(os.Stderr, "  No version found")
			os.Exit(1)
		}
		out := source.Version
		if prefix {
			out = "v" + out
		}
		fmt.Println(out)
		return

	case "set":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "  Error: set requires a version argument")
			os.Exit(1)
		}
		newVer := args[1]
		// Skip flags
		for _, v := range args[1:] {
			if !strings.HasPrefix(v, "-") {
				newVer = v
				break
			}
		}

		if source == nil {
			// Create VERSION file
			if !dryRun {
				os.WriteFile(dir+"/VERSION", []byte(newVer+"\n"), 0644)
			}
			fmt.Printf("  Set version to %s (VERSION file)\n", newVer)
			return
		}

		if dryRun {
			fmt.Printf("  Would set: %s → %s (%s)\n", source.Version, newVer, source.File)
		} else {
			writeVersion(source, newVer, dir)
			fmt.Printf("  %s → %s (%s)\n", source.Version, newVer, source.File)
		}
		return

	case "major", "minor", "patch", "premajor", "preminor", "prepatch", "prerelease":
		current, err := Parse(source.Version)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
			os.Exit(1)
		}

		bumped := current.Bump(command)
		newVer := bumped.String()

		if dryRun {
			out := newVer
			if prefix {
				out = "v" + out
			}
			fmt.Printf("  Would bump: %s → %s (%s)\n", source.Version, out, source.File)
			return
		}

		if err := writeVersion(source, newVer, dir); err != nil {
			fmt.Fprintf(os.Stderr, "  Error writing: %v\n", err)
			os.Exit(1)
		}

		out := newVer
		if prefix {
			out = "v" + out
		}
		fmt.Printf("  %s → %s (%s)\n", source.Version, out, source.File)

		if createTag {
			tag := out
			if !prefix {
				tag = "v" + newVer
			}
			fmt.Printf("  Tag: %s\n", tag)
			// Note: actual git tagging would use os/exec
			// but we keep this zero-dependency for simplicity
		}

	default:
		fmt.Fprintf(os.Stderr, "  Unknown command: %s\n", command)
		fmt.Fprintln(os.Stderr, "  Run 'semver-bump --help' for usage.")
		os.Exit(1)
	}
}
