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
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	shared "mangahub/cmd/cli/commands/shared"
	pb "mangahub/proto"
)

func HandleGrpc(args []string) {
	if len(args) < 1 {
		printUsage()
		return
	}

	subCmd := args[0]
	if subCmd != "manga" && subCmd != "progress" && subCmd != "user" {
		printUsage()
		return
	}

	flags := flag.NewFlagSet("grpc "+subCmd, flag.ExitOnError)
	addr := flags.String("addr", shared.GRPCAddr(), "gRPC server address")
	_ = flags.Parse(args[1:])
	remaining := flags.Args()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("Error connecting to gRPC server: %v\n", err)
		return
	}
	defer conn.Close()

	switch subCmd {
	case "manga":
		handleGrpcManga(pb.NewMangaServiceClient(conn), remaining)
	case "progress":
		handleGrpcProgress(pb.NewMangaServiceClient(conn), remaining)
	case "user":
		handleGrpcUser(pb.NewUserServiceClient(conn), remaining)
	}
}

func handleGrpcManga(client pb.MangaServiceClient, args []string) {
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
		resp, err := client.GetManga(ctx, &pb.GetMangaRequest{Id: *id})
		if err != nil {
			printGrpcError(err)
			return
		}
		printProto(resp.Manga)

	case "search":
		fs := flag.NewFlagSet("grpc manga search", flag.ExitOnError)
		query := fs.String("query", "", "Search query")
		limit := fs.Int("limit", 20, "Limit")
		_ = fs.Parse(args[1:])

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := client.SearchManga(ctx, &pb.SearchRequest{Query: *query, Limit: int32(*limit)})
		if err != nil {
			printGrpcError(err)
			return
		}
		printProto(resp)

	default:
		fmt.Println("Usage: mangahub grpc manga <get|search> [flags]")
	}
}

func handleGrpcProgress(client pb.MangaServiceClient, args []string) {
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
	resp, err := client.UpdateProgress(ctx, &pb.ProgressRequest{
		UserId:  resolvedUserID,
		MangaId: *mangaID,
		Chapter: int32(*chapter),
		Volume:  int32(*volume),
		Notes:   *notes,
		Force:   *force,
	})
	if err != nil {
		printGrpcError(err)
		return
	}

	printProto(resp.Result)
}

func handleGrpcUser(client pb.UserServiceClient, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub grpc user <get|library> [flags]")
		return
	}

	sub := args[0]
	switch sub {
	case "get":
		fs := flag.NewFlagSet("grpc user get", flag.ExitOnError)
		userID := fs.String("user-id", "", "User ID")
		username := fs.String("username", "", "Username")
		_ = fs.Parse(args[1:])

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := client.GetUser(ctx, &pb.GetUserRequest{UserId: *userID, Username: *username})
		if err != nil {
			printGrpcError(err)
			return
		}
		printProto(resp.User)

	case "library":
		fs := flag.NewFlagSet("grpc user library", flag.ExitOnError)
		userID := fs.String("user-id", "", "User ID")
		_ = fs.Parse(args[1:])

		if strings.TrimSpace(*userID) == "" {
			fmt.Println("--user-id is required")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := client.GetLibrary(ctx, &pb.GetLibraryRequest{UserId: *userID})
		if err != nil {
			printGrpcError(err)
			return
		}
		printProto(resp)

	default:
		fmt.Println("Usage: mangahub grpc user <get|library> [flags]")
	}
}

func printUsage() {
	fmt.Println("Usage: mangahub grpc <manga|progress|user> ...")
	fmt.Println("Examples:")
	fmt.Println("  mangahub grpc manga get --id one-piece")
	fmt.Println("  mangahub grpc manga search --query gintama")
	fmt.Println("  mangahub grpc progress update --manga-id one-piece --chapter 1095")
	fmt.Println("  mangahub grpc user get --user-id user-123")
}

func printProto(msg proto.Message) {
	if msg == nil {
		fmt.Println("<empty>")
		return
	}

	data, err := protojson.MarshalOptions{Indent: "  "}.Marshal(msg)
	if err != nil {
		fallback, _ := json.MarshalIndent(msg, "", "  ")
		fmt.Println(string(fallback))
		return
	}
	fmt.Println(string(data))
}

func printGrpcError(err error) {
	statusInfo, ok := status.FromError(err)
	if !ok {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Error (%s): %s\n", statusInfo.Code().String(), statusInfo.Message())
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
