package manga

import (
	"fmt"
)

func HandleManga(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub manga <search|list|info> [flags]")
		return
	}

	subCmd := args[0]

	switch subCmd {
	case "search":
		handleSearch(args[1:])
	case "list":
		handleList(args[1:])
	case "info":
		handleInfo(args[1:])
	default:
		fmt.Println("Unknown subcommand:", subCmd)
	}
}
