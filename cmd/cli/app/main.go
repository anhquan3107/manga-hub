package main

import (
	"fmt"
	authcmd "mangahub/cmd/cli/commands/auth"
	chatcmd "mangahub/cmd/cli/commands/chat"
	librarycmd "mangahub/cmd/cli/commands/library"
	mangacmd "mangahub/cmd/cli/commands/manga"
	notifycmd "mangahub/cmd/cli/commands/notify"
	progresscmd "mangahub/cmd/cli/commands/progress"
	servercmd "mangahub/cmd/cli/commands/server"
	synccmd "mangahub/cmd/cli/commands/sync"
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
		authcmd.HandleAuth(args)
	case "manga":
		mangacmd.HandleManga(args)
	case "server":
		servercmd.HandleServer(args)
	case "library":
		librarycmd.HandleLibrary(args)
	case "progress":
		progresscmd.HandleProgress(args)
	case "chat":
		chatcmd.HandleChat(args)
	case "sync":
		synccmd.HandleSync(args)
	case "notify":
		notifycmd.HandleNotify(args)
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
	fmt.Println("  library    Manage your library (add, list, remove, update)")
	fmt.Println("  progress   Manage your reading progress (update)")
	fmt.Println("  chat       Chat with community (join, send)")
	fmt.Println("  sync       Manage TCP synchronization (connect, disconnect, status, monitor)")
	fmt.Println("  notify     Manage UDP notifications (subscribe, unsubscribe, preferences, test)")
}
