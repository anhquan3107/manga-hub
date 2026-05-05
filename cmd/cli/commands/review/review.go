package review

import "fmt"

func HandleReview(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub review <add|list|mine|helpful> [flags]")
		return
	}

	sub := args[0]
	switch sub {
	case "add":
		reviewAdd(args[1:])
	case "list":
		reviewList(args[1:])
	case "mine":
		reviewMine(args[1:])
	case "helpful":
		reviewHelpful(args[1:])
	default:
		fmt.Println("Unknown subcommand:", sub)
	}
}
