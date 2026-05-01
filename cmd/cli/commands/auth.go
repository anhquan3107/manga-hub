package commands

import authpkg "mangahub/cmd/cli/commands/auth"

func HandleAuth(args []string) {
	authpkg.HandleAuth(args)
}
