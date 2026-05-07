package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"time"
)

func handleNotifyTest(args []string) error {
    fs := flag.NewFlagSet("test", flag.ContinueOnError)
    addr := fs.String("addr", "127.0.0.1:9091", "UDP notify server address")
    manga := fs.String("manga", "", "manga id to test notify for")
    client := fs.String("client", "", "client id (optional)")
    if err := fs.Parse(args); err != nil {
        return err
    }

    cid := *client
    if cid == "" {
        cid = registeredClientID
    }
    if cid == "" {
        return fmt.Errorf("no client id provided or registered")
    }

    return notifyTest(*addr, cid, *manga)
}

func notifyTest(addr, client, mangaID string) error {
    udpAddr, err := net.ResolveUDPAddr("udp", addr)
    if err != nil {
        return err
    }
    conn, err := net.DialUDP("udp", nil, udpAddr)
    if err != nil {
        return err
    }
    defer conn.Close()

    msg := udpClientMessage{Type: "test", Client: client, MangaID: mangaID}
    b, _ := json.Marshal(msg)
    if _, err := conn.Write(b); err != nil {
        return err
    }

    // wait for response
    if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
        return err
    }
    buf := make([]byte, 1024)
    n, _, err := conn.ReadFromUDP(buf)
    if err == nil {
        var resp udpServerMessage
        _ = json.Unmarshal(buf[:n], &resp)
        if resp.Type == "ok" {
            fmt.Println("test OK:", resp.Message)
            return nil
        }
        return fmt.Errorf("server response: %s", resp.Message)
    }

    return nil
}
