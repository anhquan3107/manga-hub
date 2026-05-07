package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

func handleSyncStatus(args []string) error {
    fs := flag.NewFlagSet("status", flag.ExitOnError)
    if err := fs.Parse(args); err != nil {
        return err
    }

    fmt.Println("TCP Sync Status:")

    conn, err := net.DialTimeout("tcp", tcpAddr, 2*time.Second)
    if err != nil {
        fmt.Println("Connection: ✗ Inactive")
        return nil
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
        if err := json.Unmarshal(data, &s); err != nil {
            fmt.Println(" Session ID: (invalid session file)")
            fmt.Println()
            fmt.Println("Sync Statistics:")
            fmt.Printf(" Messages sent: %d\n", messagesSent)
            fmt.Printf(" Messages received: %d\n", messagesRecv)
            return nil
        }

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
    return nil
}
