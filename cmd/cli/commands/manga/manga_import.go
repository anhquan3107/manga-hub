package manga

import (
	"context"
	"flag"
	"fmt"
	"mangahub/internal/data-collector/fetcher"
	"os"
	"strings"
	"time"
)

func handleImport(args []string) {
	flags := flag.NewFlagSet("manga import", flag.ExitOnError)
	var source string
	var limit int
	var seedFile string
	flags.StringVar(&source, "source", "mangadex", "API source to use (mangadex or jikan)")
	flags.IntVar(&limit, "limit", 100, "Number of manga entries to fetch")
	flags.StringVar(&seedFile, "seed-file", os.Getenv("SEED_FILE"), "Seed JSON file to update")

	if err := flags.Parse(args); err != nil {
		fmt.Println("Error parsing flags:", err)
		return
	}

	seedFile = strings.TrimSpace(seedFile)
	if seedFile == "" {
		seedFile = "./data/manga.sample.json"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	result, err := fetcher.Collect(ctx, fetcher.Options{
		SeedFile: seedFile,
		Source:   source,
		Limit:    limit,
	})
	if err != nil {
		fmt.Println("Error importing manga:", err)
		return
	}

	fmt.Printf("Imported manga from %s\n", result.Source)
	fmt.Printf("Seed file: %s\n", seedFile)
	fmt.Printf("Fetched: %d | Existing: %d | Final: %d\n", result.FetchedCount, result.ExistingCount, result.FinalCount)
	fmt.Println("Next step: restart the API server so it seeds from the updated JSON file.")
}