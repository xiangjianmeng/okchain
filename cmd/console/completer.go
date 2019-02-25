// Copyright The go-okchain Authors 2018,  All rights reserved.

package console

import (
	"strings"

	"github.com/c-bata/go-prompt"
)

func Completer(d prompt.Document) []prompt.Suggest {
	if d.TextBeforeCursor() == "" {
		return []prompt.Suggest{}
	}
	args := strings.Split(d.TextBeforeCursor(), ".")
	w := d.GetWordBeforeCursor()
	// If PIPE is in text before the cursor, returns empty suggestions.
	for i := range args {
		if args[i] == "|" {
			return []prompt.Suggest{}
		}
	}

	if strings.HasSuffix(w, ".") {
		return optionCompleter(args, strings.HasSuffix(w, "."))
	}

	// If word before the cursor starts with "-", returns CLI flag options.
	// if strings.HasPrefix(w, "-") {
	// return optionCompleter(args, strings.HasPrefix(w, "--"))
	// }
	// Return suggestions for option
	if suggests, found := completeOptionArguments(d); found {
		return suggests
	}

	return argumentsCompleter(excludeOptions(args))
}

var commands = []prompt.Suggest{
	// Wallet command
	{Text: "wallet", Description: "wallet to manage account."},

	// Node command.
	{Text: "okc", Description: "operate Tx/Contract"},

	//Console control command
	{Text: "exit", Description: "Exit this program"},
}

func argumentsCompleter(args []string) []prompt.Suggest {
	if len(args) <= 1 {
		return prompt.FilterHasPrefix(commands, args[0], true)
	}

	first := args[0]
	switch first {
	case "get":
		second := args[1]
		if len(args) == 2 {
			subcommands := []prompt.Suggest{
				{Text: "componentstatuses"},
				// aliases
				{Text: "cs"},
			}
			return prompt.FilterHasPrefix(subcommands, second, true)
		}

		third := args[2]
		if len(args) == 3 {
			switch second {
			case "replicasets", "rs":
				return prompt.FilterContains(getServiceSuggestions(), third, true)
			}
		}
	default:
		return []prompt.Suggest{}
	}
	return []prompt.Suggest{}
}
