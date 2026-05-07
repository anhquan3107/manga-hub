package commands

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"time"
)

func handleSyncMonitor(args []string) error {
    fs := flag.NewFlagSet("monitor", flag.ExitOnError)
    if err := fs.Parse(args); err != nil {
        return err
    }
    fmt.Println("Monitoring real-time progress updates... (Press CTRL+C to exit)")
    return syncMonitor()
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
    data, err := json.Marshal(hello)
    if err != nil {
        return err
    }
    if _, err := conn.Write(append(data, '\n')); err != nil {
        return err
    }

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
