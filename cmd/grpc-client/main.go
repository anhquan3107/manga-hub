package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"

	"mangahub/internal/grpcjson"
	"mangahub/internal/grpcserver"
)

func main() {
	addr := flag.String("addr", "localhost:9092", "gRPC server address")
	method := flag.String("method", "search", "Method to call: get|search|progress")
	id := flag.String("id", "one-piece", "Manga ID for get")
	query := flag.String("query", "", "Query text for search")
	genre := flag.String("genre", "", "Genre filter for search")
	status := flag.String("status", "", "Status filter for search/progress")
	limit := flag.Int("limit", 5, "Result limit for search")
	userID := flag.String("user", "user-demo", "User ID for progress")
	mangaID := flag.String("manga", "one-piece", "Manga ID for progress")
	chapter := flag.Int("chapter", 1, "Current chapter for progress")
	flag.Parse()

	codec := grpcjson.Codec{}
	encoding.RegisterCodec(codec)

	conn, err := grpc.Dial(
		*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(codec)),
	)
	if err != nil {
		log.Fatalf("dial grpc: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch *method {
	case "get":
		resp := &grpcserver.MangaResponse{}
		err = conn.Invoke(ctx, "/mangahub.MangaService/GetManga", &grpcserver.GetMangaRequest{ID: *id}, resp)
		if err != nil {
			log.Fatalf("GetManga failed: %v", err)
		}
		fmt.Printf("GetManga: %s by %s (status=%s)\n", resp.Manga.Title, resp.Manga.Author, resp.Manga.Status)
	case "search":
		resp := &grpcserver.SearchResponse{}
		err = conn.Invoke(ctx, "/mangahub.MangaService/SearchManga", &grpcserver.SearchRequest{
			Query:  *query,
			Genre:  *genre,
			Status: *status,
			Limit:  *limit,
		}, resp)
		if err != nil {
			log.Fatalf("SearchManga failed: %v", err)
		}
		fmt.Printf("SearchManga returned %d items\n", len(resp.Items))
		for i, item := range resp.Items {
			if i >= 10 {
				break
			}
			fmt.Printf("%d. %s (%s)\n", i+1, item.Title, item.Status)
		}
	case "progress":
		resp := &grpcserver.ProgressResponse{}
		err = conn.Invoke(ctx, "/mangahub.MangaService/UpdateProgress", &grpcserver.ProgressRequest{
			UserID:         *userID,
			MangaID:        *mangaID,
			CurrentChapter: *chapter,
			Status:         *status,
		}, resp)
		if err != nil {
			log.Fatalf("UpdateProgress failed: %v", err)
		}
		fmt.Printf("UpdateProgress: success=%t message=%s\n", resp.Success, resp.Message)
	default:
		log.Fatalf("unknown method %q, use get|search|progress", *method)
	}
}
