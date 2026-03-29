//go:build !(darwin || linux)

package main

import "os/exec"

func configureDetachedProcess(cmd *exec.Cmd) {}
