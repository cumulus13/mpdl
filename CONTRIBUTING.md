# Contributing to mpdl

Thank you for your interest in contributing to mpdl! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. Please be respectful and constructive in all interactions.

## How to Contribute

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When creating a bug report, include:

- **Clear title and description**
- **Steps to reproduce** the issue
- **Expected behavior** vs **actual behavior**
- **Environment details** (OS, Go version, MPD version)
- **Configuration** (sanitized, remove passwords)
- **Debug output** (run with `--debug` flag)

**Example bug report:**

```markdown
### Bug: Monitor mode crashes on song change

**Environment:**
- OS: Windows 11
- Go: 1.21.5
- MPD: 0.23.12
- mpdl: v1.0.0

**Steps to reproduce:**
1. Start monitor mode: `mpdl monitor`
2. Play a song
3. Skip to next song
4. Application crashes

**Expected:** Monitor continues running
**Actual:** Application exits with error

**Debug output:**
```
[debug output here]
```

**Configuration:**
```toml
[mpd]
host = "localhost"
port = "6600"
```
```

### Suggesting Features

Feature suggestions are welcome! Please include:

- **Use case** - Why is this feature needed?
- **Proposed solution** - How should it work?
- **Alternatives** - What other solutions have you considered?
- **Additional context** - Screenshots, examples, etc.

### Pull Requests

1. **Fork** the repository
2. **Create a branch** from `develop` (not `main`)
3. **Make your changes** with clear, descriptive commits
4. **Test** your changes thoroughly
5. **Update documentation** if needed
6. **Submit a pull request** to `develop` branch

#### Pull Request Guidelines

- **One feature/fix per PR**
- **Follow existing code style**
- **Add tests** for new functionality
- **Update README/docs** if needed
- **Keep commits atomic** and well-described
- **Ensure CI passes** before requesting review

**Example commit messages:**

```
✅ Good:
feat: add playlist shuffle command
fix: resolve connection timeout in monitor mode
docs: update installation instructions for macOS
refactor: simplify path normalization logic

❌ Bad:
updated stuff
fix bug
changes
```

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- MPD server (for testing)
- Make (optional, but recommended)

### Getting Started

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/mpdl.git
cd mpdl

# Add upstream remote
git remote add upstream https://github.com/cumulus13/mpdl.git

# Create a feature branch
git checkout -b feature/my-awesome-feature develop

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run with debug
./mpdl --debug status
```

### Project Structure

```
mpdl/
├── main.go              # Main application code
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
├── README.md           # User documentation
├── CONTRIBUTING.md     # This file
├── LICENSE             # MIT license
├── CHANGELOG.md        # Version history
├── Makefile            # Build automation
├── config.toml.example # Example configuration
├── .github/
│   └── workflows/
│       └── build.yml   # CI/CD pipeline
└── .gitignore          # Git ignore rules
```

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `go vet` before committing
- Use meaningful variable names
- Add comments for complex logic
- Keep functions focused and small

**Example:**

```go
// Good: Clear, documented, focused function
// normalizePath converts a file path to MPD-compatible format
func (m *MPDClient) normalizePath(path string) string {
    path = filepath.ToSlash(path)
    musicRoot := filepath.ToSlash(m.config.MPD.MusicRoot)
    path = strings.ReplaceAll(path, musicRoot+"/", "")
    return strings.TrimPrefix(path, "/")
}

// Bad: Unclear, undocumented, too long
func doStuff(p string) string {
    // 100 lines of mixed concerns...
}
```

### Testing

- Add tests for new features
- Ensure existing tests pass
- Test on multiple platforms if possible
- Include edge cases

```bash
# Run all tests
go test -v ./...

# Run with race detector
go test -race ./...

# Run with coverage
go test -cover ./...
```

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Create release archives
make release

# Development build with race detector
make dev
```

## Commit Message Format

Use conventional commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `refactor`: Code refactoring
- `test`: Adding/updating tests
- `chore`: Maintenance tasks
- `perf`: Performance improvements
- `ci`: CI/CD changes

**Examples:**

```
feat(monitor): add keyboard shortcuts for volume control

Add '+' and '-' keys to increase/decrease volume in monitor mode.
This allows quick volume adjustments without leaving the monitor.

Closes #42
```

```
fix(playlist): resolve race condition in delete operation

The delete operation was accessing the playlist before ensuring
connection was established, causing intermittent failures.

Fixes #38
```

## Release Process

Releases are automated via GitHub Actions when tags are pushed:

1. Update `CHANGELOG.md` with changes
2. Update version in `main.go` if needed
3. Create and push a tag:
   ```bash
   git tag -a v1.1.0 -m "Release v1.1.0"
   git push upstream v1.1.0
   ```
4. GitHub Actions will build and create the release

## Documentation

When adding features:

1. Update `README.md` with usage examples
2. Update `--help` text in `main.go`
3. Add to `CHANGELOG.md`
4. Update `config.toml.example` if adding config options

## Need Help?

- 💬 [Open a discussion](https://github.com/cumulus13/mpdl/discussions)
- 📧 Email: cumulus13@gmail.com
- 🐛 [Report an issue](https://github.com/cumulus13/mpdl/issues)

## Recognition

Contributors will be recognized in:
- GitHub Contributors page
- `CHANGELOG.md` for significant contributions
- Release notes

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to mpdl! 🎵
