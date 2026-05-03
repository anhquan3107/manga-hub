package commands

import (
	"fmt"
	"os"
	"os/exec"
)

func HandleServer(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub server start")
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
	default:
		fmt.Println("Unknown subcommand:", sub)
	}
}
