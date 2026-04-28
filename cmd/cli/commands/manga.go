package commands

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
)

func HandleManga(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub manga <search|list|info> [flags]")
		return
	}

	subCmd := args[0]
	flags := flag.NewFlagSet("manga "+subCmd, flag.ExitOnError)
	var query string
	flags.StringVar(&query, "query", "", "Search query for manga")

	switch subCmd {
	case "search", "list":
		flags.Parse(args[1:])
		u, _ := url.Parse("http://localhost:8080/manga")
		if query != "" {
			q := u.Query()
			q.Set("query", query)
			u.RawQuery = q.Encode()
		}
		resp, err := http.Get(u.String())
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		fmt.Printf("Status: %s\n", resp.Status)
		printRespBody(resp.Body)

	case "info":
		if len(args) < 2 {
			fmt.Println("Usage: mangahub manga info <manga-id>")
			return
		}
		id := args[1]
		resp, err := http.Get("http://localhost:8080/manga/" + id)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		fmt.Printf("Status: %s\n", resp.Status)
		printRespBody(resp.Body)

	default:
		fmt.Println("Unknown subcommand:", subCmd)
	}
}
