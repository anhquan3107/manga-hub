package database

import (
	"context"
	"fmt"
	"time"

	"mangahub/pkg/models"
)

func (s *Store) InsertChatMessage(ctx context.Context, msg models.ChatMessage, roomID string) error {
	id := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO chat_messages (id, user_id, username, room_id, message, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id,
		msg.UserID,
		msg.Username,
		roomID,
		msg.Message,
		time.Unix(msg.Timestamp, 0),
	)
	if err != nil {
		return fmt.Errorf("insert chat message: %w", err)
	}
	return nil
}

func (s *Store) ListChatMessages(ctx context.Context, roomID string, limit int) ([]models.ChatMessage, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT user_id, username, message, CAST(strftime('%s', created_at) AS INTEGER)
		 FROM chat_messages
		 WHERE room_id = ?
		 ORDER BY created_at DESC
		 LIMIT ?`,
		roomID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query chat messages: %w", err)
	}
	defer rows.Close()

	messages := make([]models.ChatMessage, 0, limit)
	for rows.Next() {
		var msg models.ChatMessage
		if err := rows.Scan(&msg.UserID, &msg.Username, &msg.Message, &msg.Timestamp); err != nil {
			return nil, fmt.Errorf("scan chat message: %w", err)
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chat messages: %w", err)
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// SendPrivateMessage stores a private message between two users
func (s *Store) SendPrivateMessage(ctx context.Context, pm models.PrivateMessage) error {
	id := fmt.Sprintf("pm_%d", time.Now().UnixNano())
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO private_messages (id, sender_id, sender_username, recipient_id, recipient_username, message, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id,
		pm.SenderID,
		pm.SenderUsername,
		pm.RecipientID,
		pm.RecipientUsername,
		pm.Message,
		time.Unix(pm.Timestamp, 0),
	)
	if err != nil {
		return fmt.Errorf("insert private message: %w", err)
	}
	return nil
}

// ListPrivateMessages retrieves private messages received by a user
func (s *Store) ListPrivateMessages(ctx context.Context, recipientID string, limit int) ([]models.PrivateMessage, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT sender_id, sender_username, recipient_id, recipient_username, message, CAST(strftime('%s', created_at) AS INTEGER)
		 FROM private_messages
		 WHERE recipient_id = ?
		 ORDER BY created_at DESC
		 LIMIT ?`,
		recipientID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query private messages: %w", err)
	}
	defer rows.Close()

	messages := make([]models.PrivateMessage, 0, limit)
	for rows.Next() {
		var pm models.PrivateMessage
		if err := rows.Scan(&pm.SenderID, &pm.SenderUsername, &pm.RecipientID, &pm.RecipientUsername, &pm.Message, &pm.Timestamp); err != nil {
			return nil, fmt.Errorf("scan private message: %w", err)
		}
		messages = append(messages, pm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate private messages: %w", err)
	}

	// Reverse to get oldest first
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
