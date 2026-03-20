# neo-code

A minimal coding agent written in Go.

## What this build proves

This repository now demonstrates the smallest useful agent loop for a local coding assistant:

1. read code from the workspace
2. edit code in the workspace
3. run commands in the workspace
4. return the result to the user

## Architecture

The implementation uses a small agent-oriented layout inspired by a larger Go service skeleton:

- `cmd/neo-code`: CLI entrypoint
- `internal/agent`: runtime, session state, and result formatting
- `internal/tools`: file and command tools
- `internal/workspace`: workspace-safe file operations

## Run

```bash
go run .
```

## Commands

- `/read <path>`: read a file from the workspace
- `/write <path> <content>`: overwrite a file with inline content
- `/replace <path> <old> <new>`: replace the first matching text in a file
- `/run <command>`: run a shell command inside the workspace
- `/result`: print the latest tool result
- `/status`: print the workspace root
- `/help`: print help
- `/exit`: exit the CLI

## Examples

```text
/read README.md
/replace README.md "minimal CLI agent" "minimal coding agent"
/run go test ./...
/result
```

## Notes

- File operations are restricted to the current workspace root.
- `/write` expects inline content. For content with spaces, wrap it in quotes.
- `/replace` performs a single exact-text replacement.
