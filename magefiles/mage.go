package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

var Default = Deploy

func Deploy(ctx context.Context) error {
	if err := os.MkdirAll("bin", 0755); err != nil {
		return err
	}
	command := exec.Command("go", "build", "-o", "bin/xmgr", "main.go")
	command.Env = append(os.Environ(), "CGO_ENABLED=0")
	mustCommand(command)
	mustCommand(exec.Command("scp", "bin/xmgr", "optiplex:/tmp/xmgr"))
	mustCommand(exec.Command("ssh", "optiplex", "sudo", "/tmp/xmgr", "-action", "deploy"))
	return nil
}

func mustCommand(cmd *exec.Cmd) {
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		panic(err)
	}
}
