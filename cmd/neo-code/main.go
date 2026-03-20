package neocode

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"neo-code/internal/agent"
)

func Main() {
	runtime, err := agent.NewRuntime(".")
	if err != nil {
		fmt.Printf("failed to initialize runtime: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== NeoCode Minimal Agent ===")
	fmt.Printf("workspace: %s\n", runtime.WorkspaceRoot())
	fmt.Println("Proves the basic agent loop: read code, edit code, run commands, and report results.")
	fmt.Println(agent.HelpText())

	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	for {
		fmt.Print("neo-code> ")
		if !scanner.Scan() {
			fmt.Println("\nExiting NeoCode")
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if err := handleInput(ctx, runtime, line); err != nil {
			fmt.Printf("error: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("input error: %v\n", err)
	}
}

func handleInput(ctx context.Context, runtime *agent.Runtime, line string) error {
	parts, err := splitArgs(line)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "/help":
		fmt.Println(agent.HelpText())
		return nil
	case "/exit":
		fmt.Println("Exiting NeoCode")
		os.Exit(0)
		return nil
	case "/status":
		fmt.Printf("workspace: %s\n", runtime.WorkspaceRoot())
		return nil
	case "/result":
		result := runtime.LastResult()
		if result == nil {
			fmt.Println("No tool has been executed yet.")
			return nil
		}
		fmt.Println(agent.FormatResult(*result))
		return nil
	case "/read":
		if len(parts) < 2 {
			return fmt.Errorf("usage: /read <path>")
		}
		result, err := runtime.ReadFile(ctx, parts[1])
		fmt.Println(agent.FormatResult(result))
		return err
	case "/write":
		if len(parts) < 3 {
			return fmt.Errorf("usage: /write <path> <content>")
		}
		result, err := runtime.WriteFile(ctx, parts[1], strings.Join(parts[2:], " "))
		fmt.Println(agent.FormatResult(result))
		return err
	case "/replace":
		if len(parts) < 4 {
			return fmt.Errorf("usage: /replace <path> <old> <new>")
		}
		result, err := runtime.ReplaceInFile(ctx, parts[1], parts[2], strings.Join(parts[3:], " "))
		fmt.Println(agent.FormatResult(result))
		return err
	case "/run":
		if len(parts) < 2 {
			return fmt.Errorf("usage: /run <command>")
		}
		result, err := runtime.RunCommand(ctx, strings.Join(parts[1:], " "))
		fmt.Println(agent.FormatResult(result))
		return err
	default:
		return fmt.Errorf("unrecognized command %s", parts[0])
	}
}

func splitArgs(input string) ([]string, error) {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)
	escaped := false

	flush := func() {
		if current.Len() > 0 {
			args = append(args, current.String())
			current.Reset()
		}
	}

	for _, r := range input {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case inQuote:
			if r == quoteChar {
				inQuote = false
				continue
			}
			current.WriteRune(r)
		case r == '\'' || r == '"':
			inQuote = true
			quoteChar = r
		case r == ' ' || r == '\t':
			flush()
		default:
			current.WriteRune(r)
		}
	}

	if escaped {
		return nil, fmt.Errorf("unfinished escape sequence")
	}
	if inQuote {
		return nil, fmt.Errorf("unterminated quote")
	}
	flush()
	return args, nil
}
