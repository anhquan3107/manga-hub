package tcp

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"mangahub/pkg/models"
)

type RemoteBroadcaster struct {
	addr string
	conn net.Conn
	mu   sync.Mutex
}

func NewRemoteBroadcaster(addr string) *RemoteBroadcaster {
	return &RemoteBroadcaster{
		addr: addr,
	}
}

func (rb *RemoteBroadcaster) PublishProgress(update models.ProgressUpdate) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if update.Timestamp == 0 {
		update.Timestamp = time.Now().Unix()
	}

	msg := clientMessage{
		Type:      "progress_broadcast",
		Progress:  &update,
		Timestamp: update.Timestamp,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("tcp broadcaster marshal error: %v", err)
		return
	}
	payload = append(payload, '\n')

	if err := rb.ensureConnected(); err != nil {
		log.Printf("tcp broadcaster connection failed: %v", err)
		return
	}

	if err := rb.conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		log.Printf("tcp broadcaster set deadline error: %v", err)
		rb.closeConn()
		return
	}

	if _, err := rb.conn.Write(payload); err != nil {
		log.Printf("tcp broadcaster write error: %v", err)
		rb.closeConn()
		return
	}

	log.Printf("tcp broadcaster sent progress_broadcast: manga=%s user=%s chapter=%d", update.MangaID, update.UserID, update.Chapter)
}

func (rb *RemoteBroadcaster) ensureConnected() error {
	if rb.conn != nil {
		return nil
	}

	conn, err := net.Dial("tcp", rb.addr)
	if err != nil {
		return fmt.Errorf("dial tcp server: %w", err)
	}

	rb.conn = conn
	log.Printf("tcp broadcaster connected to %s", rb.addr)
	return nil
}

func (rb *RemoteBroadcaster) closeConn() {
	if rb.conn != nil {
		_ = rb.conn.Close()
		rb.conn = nil
	}
}
