package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"text/tabwriter"
	"time"
)

func HandleServer(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub server <start|health|status>")
		return
	}

	sub := args[0]
	switch sub {
	case "start":
		cmd := exec.Command("go", "run", "cmd/api-server/main.go")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Dir = ".."
		if err := cmd.Run(); err != nil {
			fmt.Println("failed to start server:", err)
		}
	case "health", "status":
		checkServerHealth()
	default:
		fmt.Println("Unknown subcommand:", sub)
	}
}

func checkServerHealth() {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("http://localhost:8080/health")
	if err != nil {
		fmt.Printf("error connecting to server: %v\n", err)
		fmt.Println("Make sure the server is running with: mangahub server start")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("unexpected response status: %d\n", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error reading response: %v\n", err)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("error parsing response: %v\n", err)
		return
	}

	// Display the health status in a formatted table
	fmt.Println("\n=== MangaHub Server Status ===\n")

	// Overall status
	status, ok := result["status"].(string)
	if !ok {
		status = "unknown"
	}
	fmt.Printf("Overall Status: %s\n\n", status)

	// Services status
	if services, ok := result["services"].(map[string]interface{}); ok {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Service\tStatus\tError")
		fmt.Fprintln(w, "-------\t------\t-----")

		serviceOrder := []string{"http_api", "grpc", "tcp", "udp"}
		for _, svc := range serviceOrder {
			if svcStatus, exists := services[svc].(map[string]interface{}); exists {
				svcStatusStr, _ := svcStatus["status"].(string)
				if svcStatusStr == "" {
					svcStatusStr = "unknown"
				}
				errMsg, _ := svcStatus["error"].(string)
				if errMsg == "" {
					errMsg = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", svc, svcStatusStr, errMsg)
			}
		}
		w.Flush()
	}

	fmt.Println("\n=== Raw Response ===")
	var buf bytes.Buffer
	json.Indent(&buf, body, "", "  ")
	fmt.Println(buf.String())
}
