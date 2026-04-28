package commands

import (
	"bufio"
	"fmt"
	"os"

	"net/url"

	"github.com/gorilla/websocket"
)

func HandleChat(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub chat <join>")
		return
	}
	subCmd := args[0]

	switch subCmd {
	case "join":
		token := loadToken()
		if token == "" {
			fmt.Println("You must login first.")
			return
		}

		u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws/chat"}
		c, _, err := websocket.DefaultDialer.Dial(u.String()+"?token="+token, nil)
		if err != nil {
			fmt.Println("Dial error:", err)
			return
		}
		defer c.Close()

		fmt.Println("Connected to chat!")

		go func() {
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					fmt.Println("\nDisconnected:", err)
					return
				}
				fmt.Printf("\n[Chat] %s\n> ", message)
			}
		}()

		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Print("> ")
			if !scanner.Scan() {
				break
			}
			msg := scanner.Text()
			if msg == "/quit" {
				break
			}
			if msg == "" {
				continue
			}

			err := c.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				fmt.Println("Write error:", err)
				return
			}
		}

	default:
		fmt.Println("Unknown subcommand:", subCmd)
	}
}
