// Copyright The go-okchain Authors 2018,  All rights reserved.

package console

import (
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
)

var yamlFileCompleter = completer.FilePathCompleter{
	IgnoreCase: true,
	Filter: func(fi os.FileInfo) bool {
		if fi.IsDir() {
			return true
		}
		if strings.HasSuffix(fi.Name(), ".yaml") || strings.HasSuffix(fi.Name(), ".yml") {
			return true
		}
		return false
	},
}

func getPreviousOption(d prompt.Document) (cmd, option string, found bool) {
	args := strings.Split(d.TextBeforeCursor(), ".")
	l := len(args)
	if l >= 2 {
		option = args[l-2]
	}
	// if strings.HasPrefix(option, "-") {
	// 	return args[0], option, true
	// }
	return args[0], option, true
}

func completeOptionArguments(d prompt.Document) ([]prompt.Suggest, bool) {
	cmd, option, found := getPreviousOption(d)
	// fmt.Println("[completeOptionArguments]", cmd, option, found)
	if !found {
		return []prompt.Suggest{}, false
	}
	switch cmd {
	case "wallet":
		switch option {
		case "newAccount":
			fmt.Println("wallet newaccount test..")
			// return yamlFileCompleter.Complete(d), true
		case "-n", "--namespace":
			// return getNameSpaceSuggestions(), true
		}
	case "okc":
		switch option {
		case "SendTransaction":
			// fmt.Println(cmd, option, found)
		}
	}
	return []prompt.Suggest{}, false
}
