package main

import (
	"bufio"
	"context"
	"fmt"
	"neo-code/ai"
	"neo-code/config"
	"neo-code/services"
	"os"
	"strings"
)

func main() {
	fmt.Println("=== NeoCode ===")
	fmt.Println("Use /switch <model> to change models, /models to list available models, /help for commands")
	// 加载配置
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

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
