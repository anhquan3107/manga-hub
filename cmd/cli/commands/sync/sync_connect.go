package commands

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

func handleSyncConnect(args []string) error {
    var userID string
    fs := flag.NewFlagSet("connect", flag.ExitOnError)
    fs.StringVar(&userID, "user-id", "", "Your user ID (from auth token)")
    if err := fs.Parse(args); err != nil {
        return err
    }
    if userID == "" {
        userID = "default-user"
    }
    return syncConnect(userID)
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
    data, err := json.Marshal(hello)
    if err != nil {
        conn.Close()
        return fmt.Errorf("failed to encode hello: %w", err)
    }
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
    connectedAt = time.Unix(resp.ConnectedAt, 0)
    session := Session{
        SessionID:   resp.SessionID,
        ConnectedAt: resp.ConnectedAt,
    }
    data, err = json.Marshal(session)
    if err != nil {
        conn.Close()
        return fmt.Errorf("failed to encode session: %w", err)
    }
    _ = os.WriteFile(".sync_session", data, 0644)
    lastHeartbeat = time.Now()
    messagesRecv++
    fmt.Println("✓ Connected successfully!")
    fmt.Println("Connection Details:")
    fmt.Printf(" Server: %s\n", tcpAddr)
    fmt.Printf(" User: %s (%s)\n", resp.Username, resp.UserID)
    fmt.Printf(" Session ID: %s\n", resp.SessionID)
    fmt.Printf(" Connected at: %s\n", connectedAt.UTC().Format("2006-01-02 15:04:05 UTC"))

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

    data, err := json.Marshal(ping)
    if err != nil {
        return err
    }
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

func handleSyncDisconnect(args []string) error {
    var userID string
    fs := flag.NewFlagSet("disconnect", flag.ExitOnError)
    fs.StringVar(&userID, "user-id", "", "User ID")
    if err := fs.Parse(args); err != nil {
        return err
    }

    if userID == "" {
        return fmt.Errorf("Please provide --user-id")
    }

    return syncDisconnect(userID)
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
    data, err := json.Marshal(hello)
    if err != nil {
        return err
    }
    if _, err := conn.Write(append(data, '\n')); err != nil {
        return err
    }

    // send disconnect message
    msg := tcpMessage{
        Type:   "disconnect",
        UserID: userID,
    }
    data, err = json.Marshal(msg)
    if err != nil {
        return err
    }
    _, err = conn.Write(append(data, '\n'))
    return err
}
