// Copyright The go-okchain Authors 2018,  All rights reserved.

package console

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Executor(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	} else if s == "quit" || s == "exit" {
		client.Close()
		fmt.Println("Bye!")
		os.Exit(0)
		return
	}
	resp, err := handleMsg(s)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	} else {
		PrettyPrint(resp)
	}
}

// handleMsg handles msg consisting of cmd and subcmd through performing JSON-RPC call.
func handleMsg(args string) (interface{}, error) {
	cmdEndIndex := strings.Index(args, ".")
	if cmdEndIndex == -1 {
		return nil, fmt.Errorf("Unknown command, [%s]", args)
	}
	cmd, remains := args[:cmdEndIndex], args[cmdEndIndex+1:]
	if !strings.Contains(remains, "(") && !strings.Contains(remains, ")") {
		return attributeCall(cmd, remains)
	}
	subCmdEndIndex := strings.Index(remains, "(")
	if subCmdEndIndex == -1 || remains[len(remains)-1] != ')' {
		return nil, fmt.Errorf("Unknown subCommand %s", args)
	}
	subcmd, remains := remains[:subCmdEndIndex], remains[subCmdEndIndex+1:len(remains)-1]
	options, err := parseSubcmdArgs(remains)
	if err != nil {
		return nil, err
	}
	for i, opt := range options {
		options[i] = strings.TrimSpace(opt)
	}
	return postRequest(cmd, subcmd, options)
}

// parseSubcmdArgs analyzes arguments of subcmd.
// all arguments are in the format of argname:arg except for initargs and callargs.
// initargs is in the format of argname:arg1, arg2... and the same to callargs.
func parseSubcmdArgs(args string) ([]string, error) {
	var options []string
	if strings.Contains(args, "initargs") {
		split := strings.LastIndex(args, ":")
		if strings.Contains(args[split+1:], "initargs") {
			return nil, fmt.Errorf("initargs and : must be placed last")
		}
		index := strings.LastIndex(args, "initargs")
		lastComIndex := strings.LastIndex(args[:index], ",")
		options = strings.Split(strings.TrimSpace(args[:lastComIndex]), ",")
		options = append(options, args[index:])
	} else if strings.Contains(args, "callargs") {
		split := strings.LastIndex(args, ":")
		if strings.Contains(args[split+1:], "callargs") {
			return nil, fmt.Errorf("callargs and : must be placed last")
		}
		index := strings.LastIndex(args, "callargs")
		lastComIndex := strings.LastIndex(args[:index], ",")
		options = strings.Split(strings.TrimSpace(args[:lastComIndex]), ",")
		options = append(options, args[index:])
	} else {
		options = strings.Split(args, ",")
	}
	return options, nil
}

func ExecuteAndGetResult(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("you need to pass the something arguments")
	}

	out := &bytes.Buffer{}
	cmd := exec.Command("/bin/sh", "-c", "okchain"+s)
	cmd.Stdin = os.Stdin
	cmd.Stdout = out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	r := string(out.Bytes())
	return r, nil
}

func Execute(s string) error {
	cmd := exec.Command("/bin/sh", "-c", "kubectl "+s)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Got error: %s\n", err.Error())
	}
	return nil
}
