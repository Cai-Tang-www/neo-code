# Agent Guidelines for go-llm-demo

## 1. Build, Lint, and Test Commands

### Building
- `go build ./...` - builds all packages
- `go build ./cmd/...` - builds only the executables (tui and server)

### Testing
- `go test ./...` - runs all tests in the repository
- To run a single test function: `go test -run TestFunctionName ./package/path`
  Example: `go test -run TestSecurityManager_Check ./internal/pkg/security`
- To run tests with verbose output: `go test -v ./...`
- To run a single test verbosely: `go test -v -run TestFunctionName ./package/path`
- To run tests with coverage: `go test -cover ./...`
- To run a specific test package: `go test ./internal/pkg/security`

### Linting and Formatting
- Format code: `go fmt ./...` or `gofmt -w <file>`
- Format and organize imports: `goimports -w <file>` or `goimports -w .`
- Verify formatting: `gofmt -d .` (should return no output)
- Note: No external linter is configured by default; formatting relies on gofmt/goimports

### Dependency Management
- Tidy modules: `go mod tidy`
- Verify dependencies: `go mod verify`
- Check for updates: `go list -u -m all` (see which have updates)
- Update dependencies: `go get -u ./...` then `go mod tidy`

## 2. Code Style Guidelines

### General
- Follow [Effective Go](https://golang.org/doc/effective_go.html) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Prioritize clarity over cleverness
- Handle errors explicitly; don't ignore them without reason

### Imports
- Group imports in this order:
  1. Standard library packages
  2. Third-party packages (github.com, golang.org, etc.)
  3. Local project packages (go-llm-demo/...)
- Separate groups with a blank line
- Within each group, sort alphabetically
- Example:
  ```
  import (
      "bufio"
      "context"
      "fmt"

      "github.com/charmbracelet/bubbletea"

      "go-llm-demo/internal/server"
  )
  ```
- Avoid relative imports (`.` or `..`); always use the full module path

### Formatting
- Run `gofmt` and `goimports` before committing
- Line length: aim for 80-120 characters; avoid extremely long lines
- Use tabs for indentation (Go standard), not spaces
- Opening braces go on the same line as the statement (if, for, func, etc.)
- Use blank lines to separate logical sections within functions
- Don't add trailing whitespace

### Types and Naming
- Package names: lowercase, single word, no underscores
- Exported names (functions, types, vars, constants): PascalCase
- Unexported names: camelCase
- Interface names:
  - Single method: method name + "er" (e.g., Reader, Writer)
  - Multiple methods: noun or noun phrase (e.g., ReadWriter, Server)
- Struct fields: exported (PascalCase), unexported (camelCase)
- Variables: camelCase; prefixed with `n` for length, `num` for count, `buf` for buffer
- Constants: exported (PascalCase), unexported (camelCase); use iota for enumerations
- Error types: end with `Error` (e.g., `ValidationError`)
- Factory functions: `New` prefix (e.g., `NewClient`)
- Getters: no `Get` prefix for struct fields (use field name directly if exported)

### Error Handling
- Check errors immediately after function calls
- Handle or return errors; don't ignore with `_` without justification
- Wrap errors with context when returning: `return fmt.Errorf("operation: %w", err)`
- Use `errors.Is()` and `errors.As()` for error inspection
- Sentinel errors: declare as `var ErrSomething = errors.New("message")`
- Don't panic in library code; panic only in main or unrecoverable situations
- Log errors appropriately (though this project doesn't have extensive logging yet)

### Comments
- Comment every exported function, type, constant, and variable
- Comment should be a complete sentence starting with the name being described
  Example: `// ParseInput parses the input string and returns a Config struct.`
- Avoid commenting bad code; rewrite it instead
- Comments should explain why, not what (unless the what is non-obvious)
- Use // for line comments; /* */ only for large blocks or generated code
- Keep comments updated when code changes

### Control Structures
- Prefer guard clauses to reduce nesting
  ```
  if err != nil {
      return err
  }
  ```
- Handle error cases first, then the happy path
- Use `for` loops with range slices/maps; avoid manual indexing when possible
- Switch statements: no fallthrough unless explicitly labeled `// fallthrough`
- Defer for resource cleanup (close files, release locks, etc.)

### Specific Conventions in This Project
- Configuration: use YAML files; load via `config.LoadAppConfig()`
- TUI components: follow bubbletea patterns in `/internal/tui/`
- Services: keep business logic in `/internal/server/service/`
- Providers: external API wrappers in `/internal/server/infra/provider/`
- Tools: implement specific capabilities in `/internal/server/infra/tools/`
- Domain models: keep pure in `/internal/server/domain/`
- Error types: define in relevant packages; prefix with package name if generic (e.g., `tool.ErrInvalidInput`)

## 3. Version Control Practices

### Commits
- Make small, focused commits
- Use conventional commit messages:
  - `feat:` for new features
  - `fix:` for bug fixes
  - `docs:` for documentation changes
  - `refactor:` for code restructuring
  - `test:` for adding/modifying tests
  - `chore:` for maintenance tasks
  - Format: `<type>: <description>` (e.g., `feat: add chat command to TUI`)
- Include motivation in commit message body if non-obvious
- Reference issues: `Fixes #123` or `Related to #456`

### Branches
- Main branch (`master` or `main`) is stable
- Feature branches: `feature/short-description`
- Bug fix branches: `fix/issue-number-description`
- Keep branches short-lived; merge via pull request

### Pull Requests
- Keep PRs focused on a single change
- Include summary of what and why
- List any breaking changes
- Mention test commands run
- Request review from relevant team members
- Ensure CI passes before merging (if applicable)

## 4. Security Guidelines

- Never commit secrets (API keys, passwords, tokens) to the repository
- Use environment variables or external secret management for sensitive data
- The `config.example.yaml` file shows the structure; never commit actual `config.yaml` with secrets
- Validate all inputs (especially from AI or external sources)
- Use the security checker (in `/internal/pkg/security/`) for tool authorization
- Keep dependencies updated to avoid known vulnerabilities
- Review code for potential injection points (command, path traversal, etc.)
- When in doubt about security implications, ask for review

## 5. Running the Application

### TUI (Terminal User Interface)
- Run: `go run ./cmd/tui`
- Requires config.yaml with AI API key
- Interactive terminal application using bubbletea

### Server
- Run: `go run ./cmd/server`
- Provides HTTP API for external clients
- Configure via config.yaml or environment variables

### Development
- To run tests while developing: `go test ./... -v`
- To format on save: configure your editor to run gofmt/goimports
- To check for race conditions: `go test -race ./...` (where applicable)

## 6. Modules and Packages

### Module Structure
- `go-llm-demo` (root)
  - `cmd/` - main applications (tui, server)
  - `internal/` - private application code
    - `pkg/` - reusable libraries (security, etc.)
    - `server/` - server-specific code
      - `domain/` - business logic entities
      - `infra/` - infrastructure (providers, tools, repositories)
      - `service/` - application services
      - `transport/` - network handlers (HTTP, etc.)
    - `tui/` - terminal user interface
      - `core/` - model, view, update logic
      - `infra/` - TUI infrastructure (API client)
  - `api/` - external API definitions (protobuf)
  - `config/` - configuration loading
  - `configs/` - example configurations
  - `data/` - data files (if any)
  - `docs/` - documentation
  - `scripts/` - helper scripts
  - `security/` - security policies (yaml files)
  - `test/` - integration/test helpers

### Dependency Rules
- No circular dependencies between packages
- Dependencies flow inward: cmd → internal → pkg
- internal packages should not depend on each other across domains
- Prefer composition over inheritance
- Interfaces define contracts; implementations live where used

## 7. Additional Notes

- This project uses Go 1.26.1+; ensure your toolchain is compatible
- The TUI requires a terminal that supports ANSI colors
- For contributions: fork, create branch, make changes, test, submit PR
- Thank you for contributing to go-llm-demo!
