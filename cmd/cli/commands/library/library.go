package library

import "fmt"

func HandleLibrary(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub library <add|list|remove|update> [flags]")
		return
	}
	sub := args[0]
	switch sub {
	case "add":
		libraryAdd(args[1:])
	case "list":
		libraryList(args[1:])
	case "remove":
		libraryRemove(args[1:])
	case "update":
		libraryUpdate(args[1:])
	default:
		fmt.Println("Unknown subcommand:", sub)
	}
}
