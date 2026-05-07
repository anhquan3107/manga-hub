package commands

import (
	"fmt"
)

// HandleSync dispatches subcommands for TCP sync: connect, disconnect, status, monitor
func HandleSync(args []string) {
    if len(args) < 1 {
        fmt.Println("Usage: mangahub sync <connect|disconnect|status|monitor> [flags]")
        return
    }

    subCmd := args[0]
    switch subCmd {
    case "connect":
        // delegate to connect handler which parses its own flags
        if err := handleSyncConnect(args[1:]); err != nil {
            fmt.Println("Error:", err)
        }
    case "disconnect":
        if err := handleSyncDisconnect(args[1:]); err != nil {
            fmt.Println("Error:", err)
        }
    case "status":
        if err := handleSyncStatus(args[1:]); err != nil {
            fmt.Println("Error:", err)
        }
    case "monitor":
        if err := handleSyncMonitor(args[1:]); err != nil {
            fmt.Println("Error:", err)
        }
    default:
        fmt.Println("Unknown subcommand:", subCmd)
    }
}
