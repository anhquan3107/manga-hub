package commands

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
)

func HandleAuth(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub auth <register|login> [flags]")
		return
	}

	subCmd := args[0]
	flags := flag.NewFlagSet("auth "+subCmd, flag.ExitOnError)
	var username, email, password string
	flags.StringVar(&username, "username", "", "Your username")
	flags.StringVar(&email, "email", "", "Your email")
	flags.StringVar(&password, "password", "", "Your password")
	flags.Parse(args[1:])

	switch subCmd {
	case "register":
		if username == "" || email == "" || password == "" {
			fmt.Println("Username, email, and password are required.")
			return
		}
		data, _ := json.Marshal(map[string]string{
			"username": username,
			"email":    email,
			"password": password,
		})
		resp, err := http.Post("http://localhost:8080/auth/register", "application/json", bytes.NewBuffer(data))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		fmt.Printf("Status: %s\n", resp.Status)
		printRespBody(resp.Body)

	case "login":
		if username == "" || password == "" {
			fmt.Println("Username and password are required.")
			return
		}
		data, _ := json.Marshal(map[string]string{
			"username": username,
			"password": password,
		})
		resp, err := http.Post("http://localhost:8080/auth/login", "application/json", bytes.NewBuffer(data))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			var res map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&res)
			token := res["token"].(string)
			saveToken(token)
			fmt.Println("Login successful. Token saved.")
		} else {
			fmt.Printf("Login failed: %s\n", resp.Status)
			printRespBody(resp.Body)
		}
	default:
		fmt.Println("Unknown subcommand:", subCmd)
	}
}
