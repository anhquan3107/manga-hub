package commands

import (
	"fmt"
)

// HandleNotify dispatches notify subcommands: subscribe, unsubscribe, preferences, test
func HandleNotify(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("notify requires a subcommand: subscribe|unsubscribe|preferences|test")
    }

    switch args[0] {
    case "subscribe":
        return handleNotifySubscribe(args[1:])
    case "unsubscribe":
        return handleNotifyUnsubscribe(args[1:])
    case "preferences":
        return handleNotifyPreferences(args[1:])
    case "test":
        return handleNotifyTest(args[1:])
    default:
        return fmt.Errorf("unknown notify subcommand: %s", args[0])
    }
}
