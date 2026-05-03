package commands

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"mangahub/pkg/models"
)

type tcpMessage struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	MangaID   string `json:"manga_id,omitempty"`
	Chapter   int    `json:"chapter,omitempty"`
}

type tcpResponse struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	Progress  *models.ProgressUpdate `json:"progress,omitempty"`
	Username    string `json:"username,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	ConnectedAt int64  `json:"connected_at,omitempty"`
	Devices     int    `json:"devices,omitempty"`
	Timestamp int64  `json:"timestamp"`
}
type Session struct {
	SessionID   string `json:"session_id"`
	ConnectedAt int64  `json:"connected_at"`
}
var (
	tcpConn        net.Conn
	tcpAddr        = "localhost:9090"
	sessionID      string
	connectedAt    time.Time
	lastHeartbeat  time.Time
	messagesSent   int
	messagesRecv   int
)

func HandleSync(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub sync <connect|disconnect|status|monitor> [flags]")
		return
	}

	subCmd := args[0]
	flags := flag.NewFlagSet("sync "+subCmd, flag.ExitOnError)

	switch subCmd {
	case "connect":
		var userID string
		flags.StringVar(&userID, "user-id", "", "Your user ID (from auth token)")
		flags.Parse(args[1:])

		if userID == "" {
			userID = "default-user"
		}

		if err := syncConnect(userID); err != nil {
			fmt.Printf("Error connecting: %v\n", err)
		}

	case "disconnect":
		var userID string
		flags.StringVar(&userID, "user-id", "", "User ID")
		flags.Parse(args[1:])

		if userID == "" {
			fmt.Println("Please provide --user-id")
			return
		}

		if err := syncDisconnect(userID); err != nil {
			fmt.Printf("Disconnect error: %v\n", err)
		return
		}

		fmt.Println("✓ Disconnect request sent")

	case "status":
		flags.Parse(args[1:])

		fmt.Println("TCP Sync Status:")

		conn, err := net.DialTimeout("tcp", tcpAddr, 2*time.Second)
		if err != nil {
			fmt.Println("Connection: ✗ Inactive")
			return
		}
		conn.Close()

		fmt.Println("Connection: ✓ Active")
		fmt.Printf(" Server: %s\n", tcpAddr)

		// Uptime
		if !connectedAt.IsZero() {
			uptime := time.Since(connectedAt)
			fmt.Printf(" Uptime: %s\n", uptime.Truncate(time.Second))
		}

		// Heartbeat
		if !lastHeartbeat.IsZero() {
			fmt.Printf(" Last heartbeat: %s ago\n",
				time.Since(lastHeartbeat).Truncate(time.Second))
		}

		fmt.Println()
		fmt.Println("Session Info:")

		data, err := os.ReadFile(".sync_session")
		if err == nil {
			var s Session
			json.Unmarshal(data, &s)

			fmt.Printf(" Session ID: %s\n", s.SessionID)

			uptime := time.Since(time.Unix(s.ConnectedAt, 0))
			fmt.Printf(" Uptime: %s\n", uptime.Truncate(time.Second))
		} else {
			fmt.Println(" Session ID: (not connected)")
		}

		fmt.Println()
		fmt.Println("Sync Statistics:")
		fmt.Printf(" Messages sent: %d\n", messagesSent)
		fmt.Printf(" Messages received: %d\n", messagesRecv)

	case "monitor":
		flags.Parse(args[1:])
		fmt.Println("Monitoring real-time progress updates... (Press CTRL+C to exit)")
		if err := syncMonitor(); err != nil {
			fmt.Printf("Monitoring error: %v\n", err)
		}

	default:
		fmt.Println("Unknown subcommand:", subCmd)
	}
}

func syncConnect(userID string) error {
	fmt.Printf("Connecting to TCP server at %s...\n", tcpAddr)

	conn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Send hello message to register
	hello := tcpMessage{
		Type:   "hello",
		UserID: userID,
	}
	data, _ := json.Marshal(hello)
	if _, err := conn.Write(append(data, '\n')); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send hello: %w", err)
	}

	// Read hello ack
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		conn.Close()
		return fmt.Errorf("failed to receive ack")
	}

	var resp tcpResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		conn.Close()
		return fmt.Errorf("invalid response: %w", err)
	}

	if resp.Type == "error" {
		conn.Close()
		return fmt.Errorf("server error: %s", resp.Error)
	}

	tcpConn = conn
	sessionID = resp.SessionID
	connectedAt = time.Unix(resp.ConnectedAt, 0)
	session := Session{
		SessionID:   resp.SessionID,
		ConnectedAt: resp.ConnectedAt,
	}
	data, _ = json.Marshal(session)
	_ = os.WriteFile(".sync_session", data, 0644)
	lastHeartbeat = time.Now()
	messagesRecv++
	fmt.Println("✓ Connected successfully!")
	fmt.Println("Connection Details:")
	fmt.Printf(" Server: %s\n", tcpAddr)
	fmt.Printf(" User: %s (%s)\n", resp.Username, resp.UserID)
	fmt.Printf(" Session ID: %s\n", resp.SessionID)
	fmt.Printf(" Connected at: %s\n",
    connectedAt.UTC().Format("2006-01-02 15:04:05 UTC"))

	fmt.Println()
	fmt.Println("Sync Status:")
	fmt.Println(" Auto-sync: enabled")
	fmt.Println(" Conflict resolution: last_write_wins")
	fmt.Printf(" Devices connected: %d\n", resp.Devices)
	fmt.Println()
	fmt.Println("Checking connection quality...")

	if err := syncPing(); err != nil {
		fmt.Println(" Ping failed:", err)
	} else {
	fmt.Println("✓ Connection is healthy")
	}
	go func() {
    scanner := bufio.NewScanner(conn)
    for scanner.Scan() {
        messagesRecv++
        lastHeartbeat = time.Now()
    }
	}()

	// 👇 BLOCK AT THE VERY END
	select {}
}

func syncPing() error {
	if tcpConn == nil {
		return fmt.Errorf("not connected")
	}

	start := time.Now()

	ping := tcpMessage{
		Type:      "ping",
		RequestID: fmt.Sprintf("ping-%d", time.Now().Unix()),
	}

	data, _ := json.Marshal(ping)
	if _, err := tcpConn.Write(append(data, '\n')); err != nil {
		return err
	}
	messagesSent++

	scanner := bufio.NewScanner(tcpConn)
	if !scanner.Scan() {
		return fmt.Errorf("no response")
	}

	messagesRecv++

	var resp tcpResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return err
	}

	if resp.Type == "pong" {
		rtt := time.Since(start)
		lastHeartbeat = time.Now()

		fmt.Printf("Network Quality: RTT %d ms\n", rtt.Milliseconds())
		return nil
	}

	return fmt.Errorf("unexpected response: %s", resp.Type)
}

func syncMonitor() error {
	fmt.Printf("Connecting to TCP server at %s...\n", tcpAddr)

	conn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// send hello
	hello := tcpMessage{
		Type:   "hello",
		UserID: "monitor-user",
	}
	data, _ := json.Marshal(hello)
	conn.Write(append(data, '\n'))

	scanner := bufio.NewScanner(conn)

	// read hello ack
	if scanner.Scan() {
		fmt.Println("✓ Connected. Listening for updates...")
	}

	for scanner.Scan() {
		var resp tcpResponse
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			continue
		}

		if (resp.Type == "progress_broadcast" || resp.Type == "ack") && resp.Progress != nil {
			fmt.Printf("[UPDATE] user=%s manga=%s chapter=%d at %s\n",
				resp.Progress.UserID,
				resp.Progress.MangaID,
				resp.Progress.Chapter,
				time.Unix(resp.Progress.Timestamp, 0).Format("15:04:05"),
			)
			continue
		}

		if resp.Type == "broadcast" {
			fmt.Printf("[BROADCAST] %s\n", resp.Message)
		}
	}

	return scanner.Err()
}
func syncDisconnect(userID string) error {
	conn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// send hello first (like all TCP interactions)
	hello := tcpMessage{
		Type:   "hello",
		UserID: userID,
	}
	data, _ := json.Marshal(hello)
	conn.Write(append(data, '\n'))

	// send disconnect message
	msg := tcpMessage{
		Type:   "disconnect",
		UserID: userID,
	}
	data, _ = json.Marshal(msg)
	_, err = conn.Write(append(data, '\n'))
	return err
}