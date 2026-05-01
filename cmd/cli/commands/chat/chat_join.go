package chat

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	shared "mangahub/cmd/cli/commands/shared"
)

func handleChatJoin(args []string) {
	fs := flag.NewFlagSet("chat join", flag.ContinueOnError)
	mangaID := fs.String("manga-id", "", "Manga ID to join discussion")

	if err := fs.Parse(args); err != nil {
		return
	}

	token := shared.LoadToken()
	if token == "" {
		fmt.Println("✗ You must login first.")
		fmt.Println("  Try: mangahub auth login --username <username>")
		return
	}

	roomID := "general"
	if *mangaID != "" {
		roomID = *mangaID
	}

	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws/chat"}
	q := u.Query()
	q.Set("token", token)
	q.Set("room", roomID)
	u.RawQuery = q.Encode()

	fmt.Printf("Connecting to WebSocket chat server at ws://localhost:8080/ws/chat...\n")

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Printf("✗ Connection failed: %v\n", err)
		return
	}
	defer conn.Close()

	roomName := "General Chat"
	if *mangaID != "" {
		roomName = fmt.Sprintf("%s Discussion", strings.Title(*mangaID))
	}

	fmt.Printf("✓ Connected to %s\n", roomName)
	fmt.Printf("Chat Room: #%s\n", roomID)
	fmt.Println("Connected users: 1")
	fmt.Println("Your status: Online")
	fmt.Println("\nRecent messages:")
	fmt.Println("───────────────────────────────────────────────────────────────")

	messages := make(chan interface{}, 10)
	errors := make(chan error, 1)

	go func() {
		for {
			var msg map[string]interface{}
			if err := conn.ReadJSON(&msg); err != nil {
				errors <- err
				return
			}
			messages <- msg
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("\njohndoe> ")

	for {
		select {
		case <-errors:
			fmt.Println("\n✗ Connection lost")
			return
		case msg := <-messages:
			if msgData, ok := msg.(map[string]interface{}); ok {
				username, _ := msgData["username"].(string)
				message, _ := msgData["message"].(string)
				timestamp, _ := msgData["timestamp"].(float64)
				msgTime := time.Unix(int64(timestamp), 0).Format("15:04")
				fmt.Printf("\n[%s] %s: %s\n", msgTime, username, message)
				fmt.Print("johndoe> ")
			}
		default:
			if scanner.Scan() {
				input := scanner.Text()
				if strings.HasPrefix(input, "/") {
					handleChatCommand(input, conn, roomID)
					fmt.Print("johndoe> ")
					continue
				}

				if input != "" {
					msgPayload := map[string]interface{}{"message": input}
					if err := conn.WriteJSON(msgPayload); err != nil {
						fmt.Printf("✗ Failed to send message: %v\n", err)
						return
					}
				}
				fmt.Print("johndoe> ")
			}
		}
	}
}
