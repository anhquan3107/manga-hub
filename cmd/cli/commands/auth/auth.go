package auth

import (
	"flag"
	"fmt"
)

func HandleAuth(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub auth <register|login|logout|status|change-password> [flags]")
		return
	}

	subCmd := args[0]
	flags := flag.NewFlagSet("auth "+subCmd, flag.ExitOnError)
	var username, email string
	flags.StringVar(&username, "username", "", "Your username")
	flags.StringVar(&email, "email", "", "Email address")
	flags.Parse(args[1:])

	switch subCmd {
	case "register":
		handleRegister(username, email)
	case "login":
		handleLogin(username)
	case "logout":
		handleLogout()
	case "change-password":
		handleChangePassword()
	case "status":
		handleStatus()
	default:
		fmt.Println("Unknown subcommand:", subCmd)
	}
}
