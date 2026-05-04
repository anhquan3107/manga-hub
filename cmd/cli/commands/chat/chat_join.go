package chat

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	shared "mangahub/cmd/cli/commands/shared"
)

type chatMessageEvent struct {
	gen int
	msg map[string]interface{}
}

type chatErrorEvent struct {
	gen int
	err error
}

func handleChatJoin(args []string) {
	fs := flag.NewFlagSet("chat join", flag.ContinueOnError)
	mangaID := fs.String("manga-id", "", "Manga ID to join discussion")

	if err := fs.Parse(args); err != nil {
		return
	}

	token := strings.TrimSpace(shared.LoadToken())
	if token == "" {
		fmt.Println("✗ You must login first.")
		fmt.Println("  Try: mangahub auth login --username <username>")
		return
	}

	// Resolve current username from server (GET /users/me). Fallback to session ID.
	username := shared.GetSessionID()
	reqUser, _ := http.NewRequest(http.MethodGet, shared.APIURL("/users/me"), nil)
	reqUser.Header.Set("Authorization", "Bearer "+token)
	if respUser, err := http.DefaultClient.Do(reqUser); err == nil {
		defer respUser.Body.Close()
		if respUser.StatusCode == 200 {
			var u struct{ ID, Username, Email string }
			_ = json.NewDecoder(respUser.Body).Decode(&u)
			if u.Username != "" {
				username = u.Username
			}
		}
	}

	roomID := "general"
	if *mangaID != "" {
		roomID = *mangaID
	}

	dialRoom := func(targetRoom string) (*websocket.Conn, *http.Response, error) {
		u, err := url.Parse(shared.WebSocketURL("/ws/chat"))
		if err != nil {
			return nil, nil, err
		}
		q := u.Query()
		q.Set("token", token)
		q.Set("room", targetRoom)
		u.RawQuery = q.Encode()
		return websocket.DefaultDialer.Dial(u.String(), nil)
	}

	fmt.Printf("Connecting to WebSocket chat server at %s...\n", shared.WebSocketURL("/ws/chat"))

	conn, _, err := dialRoom(roomID)
	if err != nil {
		fmt.Printf("✗ Connection failed: %v\n", err)
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	roomName := chatRoomLabel(roomID)

	fmt.Printf("✓ Connected to %s\n", roomName)
	fmt.Printf("Chat Room: #%s\n", roomID)
	fmt.Println("Your status: Online")
	if history, err := fetchRoomHistory(roomID, 50); err == nil {
		printRoomHistory(roomID, history)
	} else {
		fmt.Printf("\n(History unavailable: %v)\n", err)
	}
	fmt.Println("───────────────────────────────────────────────────────────────")

	messages := make(chan chatMessageEvent, 16)
	errors := make(chan chatErrorEvent, 4)
	readerGen := 1

	startReader := func(c *websocket.Conn, gen int) {
		go func() {
			for {
				var msg map[string]interface{}
				if err := c.ReadJSON(&msg); err != nil {
					errors <- chatErrorEvent{gen: gen, err: err}
					return
				}
				messages <- chatMessageEvent{gen: gen, msg: msg}
			}
		}()
	}

	// Reader goroutine: reads messages from websocket and forwards to channel
	startReader(conn, readerGen)

	// Input goroutine: reads stdin lines and forwards to inputChan
	inputChan := make(chan string)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			inputChan <- scanner.Text()
		}
		close(inputChan)
	}()

	fmt.Printf("\n%s> ", username)

	for {
		select {
		case evt := <-errors:
			if evt.gen != readerGen {
				continue
			}
			fmt.Println("\n✗ Connection lost")
			return
		case evt := <-messages:
			if evt.gen != readerGen {
				continue
			}
			if msgData := evt.msg; msgData != nil {
				sender, _ := msgData["username"].(string)
				message, _ := msgData["message"].(string)
				timestamp, _ := msgData["timestamp"].(float64)
				msgTime := time.Unix(int64(timestamp), 0).Format("15:04")
				fmt.Printf("\n[%s] %s: %s\n", msgTime, sender, message)
				fmt.Printf("%s> ", username)
			}
		case input, ok := <-inputChan:
			if !ok {
				// stdin closed
				fmt.Println("\nInput closed")
				return
			}
			if strings.HasPrefix(input, "/") {
				nextRoom, shouldSwitch := handleChatCommand(input, roomID)
				if shouldSwitch {
					nextRoom = strings.TrimSpace(nextRoom)
					if nextRoom == "" {
						fmt.Println("✗ Usage: /manga <manga-id>")
						fmt.Printf("%s> ", username)
						continue
					}
					if nextRoom == roomID {
						fmt.Printf("✓ Already in #%s\n", roomID)
						fmt.Printf("%s> ", username)
						continue
					}

					nextConn, _, switchErr := dialRoom(nextRoom)
					if switchErr != nil {
						fmt.Printf("✗ Failed to switch room: %v\n", switchErr)
						fmt.Printf("%s> ", username)
						continue
					}

					oldConn := conn
					conn = nextConn
					roomID = nextRoom
					readerGen++
					startReader(conn, readerGen)
					_ = oldConn.Close()

					roomLabel := chatRoomLabel(roomID)
					fmt.Printf("✓ Switched to %s\n", roomLabel)
					fmt.Printf("Chat Room: #%s\n", roomID)
					if history, err := fetchRoomHistory(roomID, 50); err == nil {
						printRoomHistory(roomID, history)
					} else {
						fmt.Printf("(History unavailable: %v)\n", err)
					}
				}
				fmt.Printf("%s> ", username)
				continue
			}

			if input != "" {
				msgPayload := map[string]interface{}{"message": input}
				if err := conn.WriteJSON(msgPayload); err != nil {
					fmt.Printf("✗ Failed to send message: %v\n", err)
					return
				}
			}
			fmt.Printf("%s> ", username)
		}
	}
}
