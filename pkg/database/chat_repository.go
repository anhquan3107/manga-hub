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
