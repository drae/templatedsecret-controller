// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Kapp is now implemented using kubectl to maintain backward compatibility
// with existing tests while removing the dependency on Carvel tools
type Kapp struct {
	t         *testing.T
	namespace string
	l         Logger
}

type RunOpts struct {
	NoNamespace  bool
	IntoNs       bool
	AllowError   bool
	StderrWriter io.Writer
	StdoutWriter io.Writer
	StdinReader  io.Reader
	CancelCh     chan struct{}
	Redact       bool
	Interactive  bool
}

func (k Kapp) Run(args []string) string {
	out, _ := k.RunWithOpts(args, RunOpts{})
	return out
}

func (k Kapp) RunWithOpts(args []string, opts RunOpts) (string, error) {
	// Convert kapp commands to kubectl commands
	if len(args) < 2 {
		k.t.Fatalf("Invalid kapp command: %v", args)
		return "", fmt.Errorf("invalid command format")
	}

	var kubectlArgs []string
	var kubectlStdin io.Reader = opts.StdinReader

	// Map kapp operations to kubectl operations
	switch args[0] {
	case "deploy":
		// Handle kapp deploy -f - -a name
		if args[1] == "-f" && args[2] == "-" {
			// For kapp deploy -f - -a name, we'll use kubectl apply
			kubectlArgs = []string{"apply"}
		}
	case "delete":
		// Handle kapp delete -a name
		if args[1] == "-a" {
			// For kapp delete -a name, we'll use kubectl delete -f -
			// This requires the caller to manage what gets deleted
			kubectlArgs = []string{"delete", "--ignore-not-found=true"}
		}
	default:
		k.t.Fatalf("Unsupported kapp command: %v", args)
		return "", fmt.Errorf("unsupported kapp command: %v", args)
	}

	if !opts.NoNamespace {
		kubectlArgs = append(kubectlArgs, []string{"-n", k.namespace}...)
	}

	if kubectlArgs[0] == "apply" || kubectlArgs[0] == "delete" {
		kubectlArgs = append(kubectlArgs, "-f", "-")
	}

	k.l.Debugf("Running '%s'...\n", k.cmdDesc(kubectlArgs, opts))

	cmdName := "kubectl"
	cmd := exec.Command(cmdName, kubectlArgs...)
	cmd.Stdin = kubectlStdin

	var stderr, stdout bytes.Buffer

	if opts.StderrWriter != nil {
		cmd.Stderr = opts.StderrWriter
	} else {
		cmd.Stderr = &stderr
	}

	if opts.StdoutWriter != nil {
		cmd.Stdout = opts.StdoutWriter
	} else {
		cmd.Stdout = &stdout
	}

	if opts.CancelCh != nil {
		go func() {
			select {
			case <-opts.CancelCh:
				cmd.Process.Signal(os.Interrupt)
			}
		}()
	}

	err := cmd.Run()
	stdoutStr := stdout.String()

	if err != nil {
		err = fmt.Errorf("Execution error: stdout: '%s' stderr: '%s' error: '%s'", stdoutStr, stderr.String(), err)

		if !opts.AllowError {
			k.t.Fatalf("Failed to successfully execute '%s': %v", k.cmdDesc(kubectlArgs, opts), err)
		}
	}

	return stdoutStr, err
}

func (k Kapp) cmdDesc(args []string, opts RunOpts) string {
	prefix := "kubectl" // Changed from kapp to kubectl
	if opts.Redact {
		return prefix + " -redacted-"
	}
	return fmt.Sprintf("%s %s", prefix, strings.Join(args, " "))
}
