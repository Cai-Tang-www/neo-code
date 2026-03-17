package services

import (
	"fmt"
	"strings"

	"neo-code/ai"
)

func HandleCommand(input string, activeModel *string) error {
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
