package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"time"

	shared "mangahub/cmd/cli/commands/shared"
)

type udpClientMessage struct {
	Type      string `json:"type"`
	ClientID  string `json:"client_id,omitempty"`
	MangaID   string `json:"manga_id,omitempty"`
	Message   string `json:"message,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

type udpServerMessage struct {
	Type      string `json:"type"`
	ClientID  string `json:"client_id,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

var udpAddr = shared.UDPAddr()
var registeredClientID = ""

func HandleNotify(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub notify <subscribe|unsubscribe|preferences|test> [flags]")
		return
	}

	subCmd := args[0]
	flags := flag.NewFlagSet("notify "+subCmd, flag.ExitOnError)

	switch subCmd {
	case "subscribe":
		var clientID string
		flags.StringVar(&clientID, "client-id", "", "Client ID for notifications")
		if err := flags.Parse(args[1:]); err != nil {
			fmt.Println("Error parsing flags:", err)
			return
		}

		if clientID == "" {
			clientID = fmt.Sprintf("cli-user-%d", time.Now().Unix())
		}

		if err := notifySubscribe(clientID); err != nil {
			fmt.Printf("Error subscribing: %v\n", err)
		}

	case "unsubscribe":
		var clientID string
		flags.StringVar(&clientID, "client-id", "", "Client ID")
		if err := flags.Parse(args[1:]); err != nil {
			fmt.Println("Error parsing flags:", err)
			return
		}

		if clientID != "" {
			registeredClientID = clientID
		}

		if registeredClientID == "" {
			fmt.Println("No active subscription. Provide --client-id")
			return
		}

		fmt.Printf("Unsubscribing client %s...\n", registeredClientID)
		registeredClientID = ""
		fmt.Println("✓ Unsubscribed from notifications")

	case "preferences":
		var clientID string
		flags.StringVar(&clientID, "client-id", "", "Client ID")
		if err := flags.Parse(args[1:]); err != nil {
			fmt.Println("Error parsing flags:", err)
			return
		}

		if clientID != "" {
			registeredClientID = clientID
		}

		if registeredClientID != "" {
			fmt.Println("Notification Preferences:")
			fmt.Printf("  Client ID: %s\n", registeredClientID)
			fmt.Println("  Status: Subscribed")
			fmt.Println("  Types: manga, chapter_release")
		} else {
			fmt.Println("No active subscription. Use 'mangahub notify subscribe' or provide --client-id")
		}

	case "test":
		var mangaID string
		var clientID string
		flags.StringVar(&mangaID, "manga-id", "test-manga", "Manga ID to test notification")
		flags.StringVar(&clientID, "client-id", "", "Client ID")
		if err := flags.Parse(args[1:]); err != nil {
			fmt.Println("Error parsing flags:", err)
			return
		}

		if clientID != "" {
			registeredClientID = clientID
		}

		if registeredClientID == "" {
			fmt.Println("Not subscribed. Please subscribe first: mangahub notify subscribe")
			return
		}

		if err := notifyTest(mangaID); err != nil {
			fmt.Printf("Error sending test notification: %v\n", err)
		}

	default:
		fmt.Println("Unknown subcommand:", subCmd)
	}
}

func notifySubscribe(clientID string) error {
	fmt.Printf("Subscribing to UDP notifications at %s...\n", udpAddr)

	udpSrvAddr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve server: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, udpSrvAddr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Send register message
	register := udpClientMessage{
		Type:      "register",
		ClientID:  clientID,
		Timestamp: time.Now().Unix(),
	}
	data, _ := json.Marshal(register)
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("failed to send register: %w", err)
	}

	// Read register ack
	buffer := make([]byte, 2048)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return fmt.Errorf("failed to receive ack: %w", err)
	}

	var resp udpServerMessage
	if err := json.Unmarshal(buffer[:n], &resp); err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}

	if resp.Type == "error" {
		return fmt.Errorf("server error: %s", resp.Error)
	}

	registeredClientID = clientID
	fmt.Println("✓ Subscribed to UDP notifications!")
	fmt.Printf("  Server: %s\n", udpAddr)
	fmt.Printf("  Client ID: %s\n", clientID)
	fmt.Printf("  Message: %s\n", resp.Message)
	return nil
}

func notifyTest(mangaID string) error {
	fmt.Printf("Sending test notification for manga: %s\n", mangaID)

	udpSrvAddr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve server: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, udpSrvAddr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Send notify message
	notify := udpClientMessage{
		Type:      "notify",
		ClientID:  registeredClientID,
		MangaID:   mangaID,
		Message:   fmt.Sprintf("Test notification for %s at %s", mangaID, time.Now().Format("15:04:05")),
		Timestamp: time.Now().Unix(),
	}
	data, _ := json.Marshal(notify)
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	fmt.Println("✓ Test notification sent!")
	fmt.Printf("  Manga ID: %s\n", mangaID)
	fmt.Printf("  Timestamp: %d\n", notify.Timestamp)
	return nil
}
