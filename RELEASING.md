# Releasing Figgo

## Versioning

Figgo follows [Semantic Versioning](https://semver.org/). The version string is
embedded at build time via `-ldflags`.

## Creating a Release

1. **Ensure CI is green on `main`.**

2. **Tag the release:**

   ```bash
   git tag -a v0.1.0 -m "v0.1.0"
   git push origin v0.1.0
   ```

3. **GitHub Actions takes over.** The `release.yml` workflow will:
   - Run the full test suite
   - Build binaries for Linux, macOS, and Windows (amd64/arm64)
   - Strip debug symbols (`-s -w`) for smaller binaries
   - Generate SHA256 checksums
   - Create a GitHub Release with auto-generated release notes

4. **Edit the release** (optional): add highlights or migration notes
   using `.github/release-template.md` as a guide.

## Version Embedding

The build injects three variables into `cmd/figgo/main.go`:

| Variable | Source | Example |
|----------|--------|---------|
| `version` | `git describe --tags` | `v0.1.0` |
| `commit` | `git rev-parse --short HEAD` | `d915e30` |
| `date` | Build timestamp (UTC) | `2026-03-31T12:00:00Z` |

These are displayed by `figgo --version`:

```
figgo version v0.1.0 (commit: d915e30, built: 2026-03-31T12:00:00Z)
```

## Local Builds

`just build` and `just install` automatically inject version info from git.
For development builds without a tag, the version defaults to the short commit
hash (e.g., `d915e30`).
