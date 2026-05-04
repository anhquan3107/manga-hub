package chat

import "fmt"

func HandleChat(args []string) {
	if len(args) < 1 {
		printChatUsage()
		return
	}

	subCmd := args[0]
	switch subCmd {
	case "join":
		handleChatJoin(args[1:])
	case "send":
		handleChatSend(args[1:])
	case "history":
		handleChatHistory(args[1:])
	default:
		fmt.Printf("Unknown chat subcommand: %s\n", subCmd)
		printChatUsage()
	}
}

func printChatUsage() {
	fmt.Println("Usage: mangahub chat <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  join                Join the chat (interactive mode)")
	fmt.Println("  send <message>      Send a message to chat")
	fmt.Println("  history             View recent chat messages")
	fmt.Println("\nOptions:")
	fmt.Println("  --manga-id <id>     Join/send/history for specific manga discussion (default: general)")
	fmt.Println("  --limit <n>         Number of history messages to show (history only, default: 50)")
}
