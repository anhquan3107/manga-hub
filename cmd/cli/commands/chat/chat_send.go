package chat

import (
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"

	shared "mangahub/cmd/cli/commands/shared"
)

func handleChatSend(args []string) {
	fs := flag.NewFlagSet("chat send", flag.ContinueOnError)
	mangaID := fs.String("manga-id", "", "Manga ID to send to (default: general)")

	if err := fs.Parse(args); err != nil {
		return
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		fmt.Println("✗ Usage: mangahub chat send \"<message>\" [--manga-id <id>]")
		return
	}

	message := strings.Join(remaining, " ")
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

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Printf("✗ Connection failed: %v\n", err)
		return
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]interface{}{"message": message}); err != nil {
		fmt.Printf("✗ Failed to send message: %v\n", err)
		return
	}

	fmt.Printf("✓ Message sent to #%s\n", roomID)
}
