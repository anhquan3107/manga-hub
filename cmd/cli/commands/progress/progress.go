package commands

import "fmt"

func HandleProgress(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub progress <update|history> [flags]")
		return
	}
	subCmd := args[0]
	switch subCmd {
	case "update":
		handleProgressUpdate(args[1:])

	case "history":
		handleProgressHistory(args[1:])

	default:
		fmt.Println("Unknown subcommand:", subCmd)
	}
}
