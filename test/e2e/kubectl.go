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

type Kubectl struct {
	t         *testing.T
	namespace string
	l         Logger
}

// These options mirror what was previously in kapp.go
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

func (k Kubectl) Run(args []string) string {
	out, _ := k.RunWithOpts(args, RunOpts{})
	return out
}

func (k Kubectl) RunWithOpts(args []string, opts RunOpts) (string, error) {
	if !opts.NoNamespace {
		args = append(args, []string{"-n", k.namespace}...)
	}

	k.l.Debugf("Running '%s'...\n", k.cmdDesc(args))

	var stderr bytes.Buffer
	var stdout bytes.Buffer

	cmd := exec.Command("kubectl", args...)
	cmd.Stderr = &stderr

	if opts.CancelCh != nil {
		go func() {
			<-opts.CancelCh
			cmd.Process.Signal(os.Interrupt)
		}()
	}

	if opts.StdoutWriter != nil {
		cmd.Stdout = opts.StdoutWriter
	} else {
		cmd.Stdout = &stdout
	}

	cmd.Stdin = opts.StdinReader

	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("execution error: stderr: '%s' error: '%s'", stderr.String(), err)

		if !opts.AllowError {
			k.t.Fatalf("Failed to successfully execute '%s': %v", k.cmdDesc(args), err)
		}
	}

	return stdout.String(), err
}

func (k Kubectl) cmdDesc(args []string) string {
	return fmt.Sprintf("kubectl %s", strings.Join(args, " "))
}

// Helper methods that replace kapp functionality

// ApplyYaml applies YAML resources via kubectl apply
func (k Kubectl) ApplyYaml(yaml string, opts RunOpts) (string, error) {
	args := []string{"apply", "-f", "-"}
	opts.StdinReader = strings.NewReader(yaml)
	return k.RunWithOpts(args, opts)
}

// DeleteYaml deletes resources described in YAML via kubectl delete
func (k Kubectl) DeleteYaml(yaml string, opts RunOpts) (string, error) {
	args := []string{"delete", "--ignore-not-found=true", "-f", "-"}
	opts.StdinReader = strings.NewReader(yaml)
	return k.RunWithOpts(args, opts)
}

// DeleteByLabel deletes all resources with a specific label
func (k Kubectl) DeleteByLabel(labelKey, labelValue string, opts RunOpts) (string, error) {
	selector := fmt.Sprintf("%s=%s", labelKey, labelValue)
	args := []string{"delete", "all", "--ignore-not-found=true", "-l", selector}
	return k.RunWithOpts(args, opts)
}
