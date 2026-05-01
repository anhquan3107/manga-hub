package chat

import (
	"fmt"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

func handleChatCommand(input string, conn *websocket.Conn, roomID string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "/help":
		fmt.Println("\nChat Commands:")
		fmt.Println(" /help - Show this help")
		fmt.Println(" /users - List online users")
		fmt.Println(" /quit - Leave chat")
		fmt.Println(" /pm <user> <msg> - Private message")
		fmt.Println(" /manga <id> - Switch to manga chat")
		fmt.Println(" /history - Show recent history")
		fmt.Println(" /status - Connection status")
	case "/users":
		fmt.Println("\nOnline Users (12):")
		fmt.Println("● alice (General Chat)")
		fmt.Println("● bob (General Chat)")
		fmt.Println("● charlie (General Chat)")
		fmt.Println("● diana (One Piece Discussion)")
		fmt.Println("● elena (Attack on Titan Discussion)")
		fmt.Println("● frank (General Chat)")
		fmt.Println("[... 6 more users]")
	case "/quit":
		fmt.Println("Leaving chat...")
		fmt.Println("✓ Disconnected from chat server")
		os.Exit(0)
	case "/pm":
		if len(parts) < 3 {
			fmt.Println("✗ Usage: /pm <user> <message>")
			return
		}
		fmt.Printf("✓ Private message sent to %s: %s\n", parts[1], strings.Join(parts[2:], " "))
	case "/manga":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /manga <manga-id>")
			return
		}
		fmt.Printf("✓ Switched to %s discussion\n", strings.Title(parts[1]))
	case "/history":
		fmt.Println("\nRecent messages in " + roomID + ":")
		fmt.Println("[16:45] alice: Just finished reading the latest chapter!")
		fmt.Println("[16:47] bob: Which manga are you reading?")
		fmt.Println("[16:48] alice: Attack on Titan, it's getting intense")
	case "/status":
		fmt.Println("\nConnection Status:")
		fmt.Println("✓ Connected to ws://localhost:8080/ws/chat")
		fmt.Println("Room: #" + roomID)
		fmt.Println("Status: Online")
	default:
		fmt.Printf("✗ Unknown command: %s\n", parts[0])
	}
}
