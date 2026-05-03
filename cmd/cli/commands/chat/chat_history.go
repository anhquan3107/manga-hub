package chat

import (
	"flag"
	"fmt"
)

func handleChatHistory(args []string) {
	fs := flag.NewFlagSet("chat history", flag.ContinueOnError)
	mangaID := fs.String("manga-id", "", "Manga ID (default: general)")
	limit := fs.Int("limit", 50, "Number of messages to load")

	if err := fs.Parse(args); err != nil {
		return
	}

	roomID := "general"
	if *mangaID != "" {
		roomID = *mangaID
	}

	if *limit <= 0 {
		fmt.Println("✗ --limit must be > 0")
		return
	}

	messages, err := fetchRoomHistory(roomID, *limit)
	if err != nil {
		fmt.Printf("✗ Failed to load chat history: %v\n", err)
		return
	}

	printRoomHistory(roomID, messages)
}
