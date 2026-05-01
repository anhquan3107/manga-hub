package commands

import librarypkg "mangahub/cmd/cli/commands/library"

func HandleLibrary(args []string) {
	librarypkg.HandleLibrary(args)
}

func HandleProgress(args []string) {
	progressHandler(args)
}
