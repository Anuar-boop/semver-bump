# semver-bump

Semantic version bumper for CI/CD pipelines. Auto-detects your version source and bumps major, minor, patch, or pre-release.

```bash
go install github.com/Anuar-boop/semver-bump@latest
```

## Quick Start

```bash
semver-bump patch           # 1.2.3 → 1.2.4
semver-bump minor           # 1.2.3 → 1.3.0
semver-bump major           # 1.2.3 → 2.0.0
semver-bump prerelease      # 1.2.3 → 1.2.4-0
semver-bump current         # Show current version
semver-bump set 2.0.0-beta  # Set exact version
```

## Auto-Detection

semver-bump automatically finds your version from:

| File | Format |
|------|--------|
| `package.json` | `"version": "1.2.3"` |
| `Cargo.toml` | `version = "1.2.3"` |
| `pyproject.toml` | `version = "1.2.3"` |
| `setup.py` | `version="1.2.3"` |
| `VERSION` | `1.2.3` |

## Commands

| Command | Example | Result |
|---------|---------|--------|
| `major` | 1.2.3 → | 2.0.0 |
| `minor` | 1.2.3 → | 1.3.0 |
| `patch` | 1.2.3 → | 1.2.4 |
| `premajor` | 1.2.3 → | 2.0.0-0 |
| `preminor` | 1.2.3 → | 1.3.0-0 |
| `prepatch` | 1.2.3 → | 1.2.4-0 |
| `prerelease` | 1.2.4-0 → | 1.2.4-1 |
| `current` | — | Show version |
| `set <ver>` | — | Set exact version |

## Options

| Flag | Description |
|------|-------------|
| `--dry-run` | Show changes without writing |
| `--prefix` | Include `v` prefix in output |
| `--dir <path>` | Project directory (default: `.`) |
| `--tag` | Create git tag after bumping |

## CI/CD Example

```yaml
# GitHub Actions
- name: Bump version
  run: |
    semver-bump patch --prefix
    git add package.json
    git commit -m "chore: bump version"
    git push
```

## Features

- Auto-detects version source (npm, Cargo, Python, VERSION file)
- Full SemVer 2.0.0 support (pre-release, build metadata)
- Dry-run mode for CI safety
- Git tag creation
- Reads and writes back to the original file
- Zero dependencies (pure Go)

## License

MIT
