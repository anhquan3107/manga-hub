package main

import (
	"fmt"
	"mangahub/cmd/cli/commands"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "auth":
		commands.HandleAuth(args)
	case "manga":
		commands.HandleManga(args)
	case "server":
		commands.HandleServer(args)
	case "library":
		commands.HandleLibrary(args)
	case "progress":
		commands.HandleProgress(args)
	case "chat":
		commands.HandleChat(args)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("MangaHub CLI")
	fmt.Println("Usage: mangahub <command> <subcommand> [flags]")
	fmt.Println("\nCommands:")
	fmt.Println("  auth       Manage authentication (register, login, logout, status)")
	fmt.Println("  manga      Manage manga (search, info)")
	fmt.Println("  library    Manage your library (add, list)")
	fmt.Println("  progress   Manage your reading progress (update)")
	fmt.Println("  chat       Join the chat")
}
