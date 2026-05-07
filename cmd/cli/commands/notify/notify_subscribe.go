package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

func handleNotifySubscribe(args []string) error {
    fs := flag.NewFlagSet("subscribe", flag.ContinueOnError)
    addr := fs.String("addr", "127.0.0.1:9091", "UDP notify server address")
    client := fs.String("client", "", "client id to register")
    if err := fs.Parse(args); err != nil {
        return err
    }

    if *client == "" {
        return fmt.Errorf("client id required")
    }

    if err := notifySubscribe(*addr, *client); err != nil {
        return err
    }

    fmt.Fprintln(os.Stdout, "subscribe request sent")
    return nil
}

func notifySubscribe(addr, client string) error {
    udpAddr, err := net.ResolveUDPAddr("udp", addr)
    if err != nil {
        return err
    }
    conn, err := net.DialUDP("udp", nil, udpAddr)
    if err != nil {
        return err
    }
    defer conn.Close()

    msg := udpClientMessage{Type: "subscribe", Client: client}
    b, _ := json.Marshal(msg)
    if _, err := conn.Write(b); err != nil {
        return err
    }

    // read optional response with deadline
    if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
        return err
    }
    buf := make([]byte, 1024)
    n, _, err := conn.ReadFromUDP(buf)
    if err == nil {
        var resp udpServerMessage
        _ = json.Unmarshal(buf[:n], &resp)
        if resp.Type == "error" {
            return fmt.Errorf("server error: %s", resp.Message)
        }
    }

    // store resolved addr global for future ops
    notifyUDPAddr = udpAddr
    registeredClientID = client
    return nil
}

func handleNotifyUnsubscribe(args []string) error {
    fs := flag.NewFlagSet("unsubscribe", flag.ContinueOnError)
    addr := fs.String("addr", "127.0.0.1:9091", "UDP notify server address")
    client := fs.String("client", "", "client id to unregister")
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

    // send unsubscribe
    if notifyUDPAddr == nil || notifyUDPAddr.String() != *addr {
        a, err := net.ResolveUDPAddr("udp", *addr)
        if err != nil {
            return err
        }
        notifyUDPAddr = a
    }

    conn, err := net.DialUDP("udp", nil, notifyUDPAddr)
    if err != nil {
        return err
    }
    defer conn.Close()

    msg := udpClientMessage{Type: "unsubscribe", Client: cid}
    b, _ := json.Marshal(msg)
    if _, err := conn.Write(b); err != nil {
        return err
    }

    registeredClientID = ""
    fmt.Fprintln(os.Stdout, "unsubscribe request sent")
    return nil
}

func handleNotifyPreferences(args []string) error {
    fs := flag.NewFlagSet("preferences", flag.ContinueOnError)
    addr := fs.String("addr", "127.0.0.1:9091", "UDP notify server address")
    client := fs.String("client", "", "client id")
    pref := fs.String("pref", "", "preference payload")
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

    if notifyUDPAddr == nil || notifyUDPAddr.String() != *addr {
        a, err := net.ResolveUDPAddr("udp", *addr)
        if err != nil {
            return err
        }
        notifyUDPAddr = a
    }

    conn, err := net.DialUDP("udp", nil, notifyUDPAddr)
    if err != nil {
        return err
    }
    defer conn.Close()

    // send a preferences message (simple payload)
    m := map[string]string{"client": cid, "pref": *pref}
    b, _ := json.Marshal(m)
    if _, err := conn.Write(b); err != nil {
        return err
    }

    fmt.Fprintln(os.Stdout, "preferences sent")
    return nil
}
