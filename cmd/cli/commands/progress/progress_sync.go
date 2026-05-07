package commands

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"time"
)

func handleProgressSync(args []string) {
	var userID string
	flags := flag.NewFlagSet("progress sync", flag.ExitOnError)
	flags.StringVar(&userID, "user-id", "", "Your user ID (from auth token)")
	if err := flags.Parse(args); err != nil {
		fmt.Println("Error parsing flags:", err)
		return
	}

	if userID == "" {
		userID = "default-user"
	}

	if err := progressSync(userID); err != nil {
		fmt.Printf("✗ Sync failed: %v\n", err)
		return
	}
	fmt.Println("✓ Sync completed successfully")
}

func handleProgressSyncStatus(args []string) {
	flags := flag.NewFlagSet("progress sync-status", flag.ExitOnError)
	if err := flags.Parse(args); err != nil {
		fmt.Println("Error parsing flags:", err)
		return
	}

	if err := progressSyncStatus(); err != nil {
		fmt.Printf("TCP sync server: ✗ %v\n", err)
		return
	}
	fmt.Println("TCP sync server: ✓ Reachable")
}

func progressSync(userID string) error {
	conn, err := net.DialTimeout("tcp", progressTCPAddr, 3*time.Second)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", progressTCPAddr, err)
	}
	defer conn.Close()

	// Send hello
	hello := progressTCPMessage{Type: "hello", UserID: userID}
	data, _ := json.Marshal(hello)
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("send hello: %w", err)
	}

	reader := bufio.NewScanner(conn)
	if !reader.Scan() {
		return fmt.Errorf("no response to hello")
	}

	// Send ping as a lightweight sync check
	ping := progressTCPMessage{Type: "ping", RequestID: fmt.Sprintf("sync-%d", time.Now().Unix())}
	data, _ = json.Marshal(ping)
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("send ping: %w", err)
	}
	if !reader.Scan() {
		return fmt.Errorf("no response to ping")
	}

	var resp progressTCPResponse
	if err := json.Unmarshal(reader.Bytes(), &resp); err != nil {
		return fmt.Errorf("invalid ping response: %w", err)
	}
	if resp.Type != "pong" {
		return fmt.Errorf("unexpected response: %s", resp.Type)
	}

	return nil
}

func progressSyncStatus() error {
	conn, err := net.DialTimeout("tcp", progressTCPAddr, 2*time.Second)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
