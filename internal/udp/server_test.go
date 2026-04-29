package udp

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

type udpTestMessage struct {
	Type      string `json:"type"`
	ClientID  string `json:"client_id"`
	MangaID   string `json:"manga_id"`
	Message   string `json:"message"`
	Error     string `json:"error"`
	Timestamp int64  `json:"timestamp"`
}

func TestUDPRegisterAndBroadcast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr := freeUDPAddr(t)
	server := NewServer(addr)
	go func() {
		_ = server.ListenAndServe(ctx)
	}()

	clientA := dialUDPClient(t, addr)
	defer clientA.Close()

	clientB := dialUDPClient(t, addr)
	defer clientB.Close()

	writeUDPMessage(t, clientA, map[string]any{
		"type":      "register",
		"client_id": "client-a",
	})
	ackA := readUDPMessage(t, clientA)
	if ackA.Type != "register_ack" {
		t.Fatalf("expected register_ack for clientA, got %s", ackA.Type)
	}

	writeUDPMessage(t, clientB, map[string]any{
		"type":      "register",
		"client_id": "client-b",
	})
	ackB := readUDPMessage(t, clientB)
	if ackB.Type != "register_ack" {
		t.Fatalf("expected register_ack for clientB, got %s", ackB.Type)
	}

	writeUDPMessage(t, clientA, map[string]any{
		"type":      "notify",
		"client_id": "client-a",
		"manga_id":  "manga-1",
		"message":   "Chapter 5 is available",
	})

	notifyA := readUDPMessage(t, clientA)
	if notifyA.Type != "notification" {
		t.Fatalf("expected notification to sender, got %s", notifyA.Type)
	}
	if notifyA.ClientID != "client-a" || notifyA.Message != "Chapter 5 is available" || notifyA.MangaID != "manga-1" {
		t.Fatalf("expected direct notification payload for sender")
	}

	notifyB := readUDPMessage(t, clientB)
	if notifyB.Type != "notification" {
		t.Fatalf("expected notification to listener, got %s", notifyB.Type)
	}
	if notifyB.ClientID != "client-a" || notifyB.Message != "Chapter 5 is available" {
		t.Fatalf("expected direct notification payload from client-a")
	}
}

func TestUDPRejectsUnregisteredNotify(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr := freeUDPAddr(t)
	server := NewServer(addr)
	go func() {
		_ = server.ListenAndServe(ctx)
	}()

	client := dialUDPClient(t, addr)
	defer client.Close()

	writeUDPMessage(t, client, map[string]any{
		"type":      "notify",
		"client_id": "client-x",
		"manga_id":  "manga-1",
		"message":   "Not allowed yet",
	})

	resp := readUDPMessage(t, client)
	if resp.Type != "error" {
		t.Fatalf("expected error, got %s", resp.Type)
	}
	if resp.Error == "" {
		t.Fatalf("expected a helpful error message")
	}
}

func freeUDPAddr(t *testing.T) string {
	t.Helper()

	l, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve udp addr: %v", err)
	}
	defer l.Close()
	return l.LocalAddr().String()
}

func dialUDPClient(t *testing.T, serverAddr string) *net.UDPConn {
	t.Helper()

	addr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		t.Fatalf("resolve udp addr: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatalf("dial udp: %v", err)
	}
	return conn
}

func writeUDPMessage(t *testing.T, conn *net.UDPConn, payload any) {
	t.Helper()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write udp payload: %v", err)
	}
}

func readUDPMessage(t *testing.T, conn *net.UDPConn) udpTestMessage {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	defer conn.SetReadDeadline(time.Time{})

	buf := make([]byte, 2048)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read udp message: %v", err)
	}

	var msg udpTestMessage
	if err := json.Unmarshal(buf[:n], &msg); err != nil {
		t.Fatalf("unmarshal udp message: %v", err)
	}
	return msg
}
