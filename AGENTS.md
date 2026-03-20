# NeoCode Agent Guidelines

This file provides instructions for agentic coding agents working on the NeoCode repository.

## Build, Lint, and Test Commands

### Building
```bash
# Build the application
go build -o neo-code .

# Run the application
go run .

# Install dependencies
go mod tidy
```

### Testing
Currently, there are no test files in the repository. When adding tests:
```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./ai
go test ./services
go test ./memory
go test ./config

# Run a single test function
go test -run TestFunctionName ./package

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...
```

### Linting
```bash
# Install golangci-lint (if not available)
# go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.0

# Run linting
golangci-lint run

# Fix linting issues automatically (when supported)
golangci-lint run --fix
```

### Formatting
```bash
# Format Go code
go fmt ./...

# Format specific file
go fmt filename.go

# Check if code is properly formatted
gofmt -l .
```

## Code Style Guidelines

### Imports
- Group imports: standard library first, then third-party, then local packages
- Each group separated by a blank line
- Local imports use the repository path: `neo-code/package`
- Sort imports alphabetically within each group
- Use gofmt/goimports for automatic formatting

Example:
```go
import (
    "context"
    "fmt"
    "strings"

    "neo-code/ai"
    "neo-code/config"
)
```

### Formatting
- Use `gofmt` as the standard formatter
- Line length: No strict limit, but aim for readability (typically 80-100 characters)
- Indentation: Use tabs (Go standard)
- Blank lines: Use to separate logical sections within functions
- Control structures: Opening brace on same line, closing brace on its own line
- Switch statements: No fallthrough by default; use `fallthrough` keyword explicitly when needed

### Types
- Define structs with clear, descriptive names
- Use embedded types when appropriate for composition
- Define interfaces that satisfy the smallest possible set of methods
- Export only what needs to be public (capitalized names)
- Use composition over inheritance
- Define zero values that are meaningful or document when they're not

### Naming Conventions
- Packages: lowercase, single word, no underscores
- Variables: camelCase, descriptive but concise
- Constants: camelCase or MixedCase for const, ALL_CAPS for iota-enums
- Functions: camelCase
- Structs: MixedCase (PascalCase)
- Interfaces: MixedCase, often ending with -er (Reader, Writer, etc.)
- Methods: camelCase, receiver name should be short (1-2 letters)
- Files: snake_case.go
- Error variables: err prefix or Err suffix (errNotFound, ErrInvalidInput)

### Error Handling
- Handle errors explicitly; don't ignore them
- Return errors early when they prevent normal function execution
- Wrap errors with context using `fmt.Errorf()` or `errors.Wrap()` when adding value
- Sentinel errors: predeclare errors for specific error conditions
- Error strings: lowercase, no punctuation unless including proper nouns
- Panic only for truly unrecoverable situations (e.g., initialization failures)
- Recover only at package boundaries, not for flow control

### Comments
- File comment: Each file should start with a comment describing its purpose
- Function comments: Describe what the function does, parameters, return values, and any side effects
- Complex code blocks: Add explanatory comments
- TODO comments: Use `// TODO:` format for tracking work that needs to be done
- Avoid obvious comments; focus on why, not what

### Specific Patterns in This Codebase
- Configuration: Uses struct tags for YAML decoding (`yaml:"key"`)
- Context: Always pass context.Context as first parameter when appropriate
- Interfaces: Define clear interfaces for providers (ChatProvider, EmbeddingProvider)
- Error checking: Check errors immediately after function calls
- Memory management: Use sync.Once for singleton initialization
- Channel usage: Close channels when done sending to prevent goroutine leaks
- String building: Use strings.Builder for efficient string concatenation
- Time handling: Use time.Now().UTC() for consistent timestamps

### Memory Package Specifics
- Store interface: Abstract storage mechanism
- Item struct: Contains all necessary fields for memory items
- Search: Uses cosine similarity for matching memories
- Serialization: JSON-based persistence

### Services Package Specifics
- Memory augmentation: Enhances chat context with relevant memories
- Provider abstraction: Supports multiple AI providers through interfaces
- Streaming: Uses channels for streaming responses from AI models

### AI Package Specifics
- Provider interfaces: Define contracts for different AI capabilities
- Message struct: Standard format for chat messages
- Factory functions: New*ProviderFromEnv for creating providers from configuration

### Config Package Specifics
- Global state: Uses package-level variables for configuration
- YAML loading: Supports reloading configuration
- Defaults: Provides sensible defaults when configuration is missing

## Directory Structure
```
neo-code/
├── main.go              # Application entry point
├── config.yaml          # Main configuration file
├── config/              # Configuration loading and structures
├── ai/                  # AI provider interfaces and implementations
├── services/            # Business logic (chat service, memory handling)
├── memory/              # Memory storage and retrieval
├── data/                # Data files (memory.json, etc.)
├── persona.txt          # Persona/prompt file
└── go.mod               # Go module definition
```

## Common Workflows
1. Adding a new AI provider:
   - Define provider interfaces in ai/ if needed
   - Implement the provider in a new file
   - Update factory functions to instantiate the new provider
   - Add configuration options if needed

2. Modifying memory handling:
   - Update memory/storage interfaces if changing storage mechanism
   - Modify Item struct if changing what's stored
   - Update search algorithms if changing matching strategy
   - Adjust configuration options as needed

3. Changing chat behavior:
   - Modify services/request.go for core chat logic
   - Update context augmentation strategies
   - Adjust memory saving/retrieval parameters
   - Update command handling in services/REPL.go if needed

## When in Doubt
- Follow existing patterns in the codebase
- Use gofmt/goimports for formatting
- Write clear, concise comments explaining why
- Handle errors explicitly
- Keep functions focused on a single responsibility
- Write tests for new functionality