package grpc

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	shared "mangahub/cmd/cli/commands/shared"
	"mangahub/internal/grpc/grpcjson"
)

type MangaRequest struct {
	Id    string `json:"id,omitempty"`
	Query string `json:"query,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type MangaResponse struct {
	Manga  any    `json:"manga,omitempty"`
	Items  any    `json:"items,omitempty"`
	Status string `json:"status,omitempty"`
	Error  string `json:"error,omitempty"`
}

type ProgressRequest struct {
	UserID  string `json:"user_id,omitempty"`
	MangaID string `json:"manga_id,omitempty"`
	Chapter int    `json:"chapter,omitempty"`
	Volume  int    `json:"volume,omitempty"`
	Notes   string `json:"notes,omitempty"`
	Force   bool   `json:"force,omitempty"`
}

type ProgressResponse struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func HandleGrpc(args []string) {
	if len(args) < 1 {
		printUsage()
		return
	}

	subCmd := args[0]
	if subCmd != "manga" && subCmd != "progress" {
		printUsage()
		return
	}

	flags := flag.NewFlagSet("grpc "+subCmd, flag.ExitOnError)
	addr := flags.String("addr", shared.GRPCAddr(), "gRPC server address")
	_ = flags.Parse(args[1:])
	remaining := flags.Args()

	conn, err := grpc.NewClient(*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(grpcjson.Codec{})),
	)
	if err != nil {
		fmt.Printf("Error connecting to gRPC server: %v\n", err)
		return
	}
	defer conn.Close()

	switch subCmd {
	case "manga":
		handleGrpcManga(conn, remaining)
	case "progress":
		handleGrpcProgress(conn, remaining)
	}
}

func handleGrpcManga(conn *grpc.ClientConn, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub grpc manga <get|search> [flags]")
		return
	}

	sub := args[0]
	switch sub {
	case "get":
		fs := flag.NewFlagSet("grpc manga get", flag.ExitOnError)
		id := fs.String("id", "", "Manga ID")
		_ = fs.Parse(args[1:])
		if strings.TrimSpace(*id) == "" {
			fmt.Println("--id is required")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var resp MangaResponse
		if err := conn.Invoke(ctx, "/mangahub.MangaHub/GetManga", &MangaRequest{Id: *id}, &resp); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		b, _ := json.MarshalIndent(resp.Manga, "", "  ")
		fmt.Println(string(b))

	case "search":
		fs := flag.NewFlagSet("grpc manga search", flag.ExitOnError)
		query := fs.String("query", "", "Search query")
		limit := fs.Int("limit", 20, "Limit")
		_ = fs.Parse(args[1:])

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var resp MangaResponse
		if err := conn.Invoke(ctx, "/mangahub.MangaHub/SearchManga", &MangaRequest{Query: *query, Limit: *limit}, &resp); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		items, _ := json.MarshalIndent(resp.Items, "", "  ")
		fmt.Println(string(items))

	default:
		fmt.Println("Usage: mangahub grpc manga <get|search> [flags]")
	}
}

func handleGrpcProgress(conn *grpc.ClientConn, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub grpc progress update --manga-id <id> --chapter <number>")
		return
	}

	sub := args[0]
	if sub != "update" {
		fmt.Println("Usage: mangahub grpc progress update --manga-id <id> --chapter <number>")
		return
	}

	fs := flag.NewFlagSet("grpc progress update", flag.ExitOnError)
	mangaID := fs.String("manga-id", "", "Manga ID")
	chapter := fs.Int("chapter", 0, "Chapter")
	volume := fs.Int("volume", 0, "Volume")
	userID := fs.String("user-id", "default-user", "User ID")
	force := fs.Bool("force", false, "Force backwards update")
	notes := fs.String("notes", "", "Notes")
	_ = fs.Parse(args[1:])

	if strings.TrimSpace(*mangaID) == "" || *chapter == 0 {
		fmt.Println("--manga-id and --chapter are required")
		return
	}

	resolvedUserID := strings.TrimSpace(*userID)
	if resolvedUserID == "" || resolvedUserID == "default-user" {
		resolvedUserID = resolveUserIDFromToken()
		if resolvedUserID == "" {
			resolvedUserID = "default-user"
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var resp ProgressResponse
	if err := conn.Invoke(ctx, "/mangahub.MangaHub/UpdateProgress", &ProgressRequest{
		UserID:  resolvedUserID,
		MangaID: *mangaID,
		Chapter: *chapter,
		Volume:  *volume,
		Notes:   *notes,
		Force:   *force,
	}, &resp); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	b, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println(string(b))
}

func printUsage() {
	fmt.Println("Usage: mangahub grpc <manga|progress> ...")
	fmt.Println("Examples:")
	fmt.Println("  mangahub grpc manga get --id one-piece")
	fmt.Println("  mangahub grpc manga search --query gintama")
	fmt.Println("  mangahub grpc progress update --manga-id one-piece --chapter 1095")
}

func resolveUserIDFromToken() string {
	token := strings.TrimSpace(shared.LoadToken())
	if token == "" {
		return ""
	}
	req, err := http.NewRequest(http.MethodGet, shared.APIURL("/users/me"), nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var user struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return ""
	}
	return strings.TrimSpace(user.ID)
}
