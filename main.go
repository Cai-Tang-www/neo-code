package main

import (
	"bufio"
	"context"
	"fmt"
	"neo-code/ai"
	"neo-code/services"
	"os"
	"strings"
)

func main() {
	fmt.Println("=== NeoCode ===")
	fmt.Println("Use /switch <model> to change models, /models to list available models, /help for commands")

	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()
	activeModel := ai.DefaultModel()
	if activeModel == "" {
		fmt.Println("Default model is not configured, cannot start")
		return
	}

	for {
		fmt.Printf("[%s] > ", activeModel)
		if !scanner.Scan() {
			fmt.Println("\nExiting NeoCode")
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			if err := services.HandleCommand(line, &activeModel); err != nil {
				fmt.Println(err)
			}
			continue
		}

		fmt.Println("Thinking...")
		messages := []ai.Message{
			{Role: "user", Content: line},
		}
		rep, err := services.Chat(ctx, messages, activeModel)
		if err != nil {
			fmt.Printf("Generation failed: %v\n", err)
			continue
		}
		for msg := range rep {
			fmt.Print(msg)
		}
		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Input error: %v\n", err)
	}
}

func handleCommand(input string, activeModel *string) error {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return nil
	}

	switch fields[0] {
	case "/switch":
		if len(fields) < 2 {
			printAvailableModels()
			return fmt.Errorf("Usage: /switch <model>")
		}
		target := fields[1]
		if !ai.IsSupportedModel(target) {
			printAvailableModels()
			return fmt.Errorf("Model %q is not supported", target)
		}
		*activeModel = target
		fmt.Printf("Switched to model %s\n", target)
	case "/models":
		printAvailableModels()
	case "/help":
		printHelp()
	default:
		fmt.Printf("Unrecognized command %s, try /help\n", fields[0])
	}
	return nil
}

func printAvailableModels() {
	fmt.Println("Available models:")
	for _, model := range ai.SupportedModels {
		fmt.Printf("  %s\n", model)
	}
}

func printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  /switch <model>  Switch the active model")
	fmt.Println("  /models          List supported models")
	fmt.Println("  /help            Show this help text")
	fmt.Println("All other input is treated as a prompt sent to the model.")
}
