// +build integration

/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"bytes"
	"syscall"

	"github.com/minishift/minishift/pkg/minikube/constants"
	instanceState "github.com/minishift/minishift/pkg/minishift/config"
)

type MinishiftRunner struct {
	CommandPath string
	CommandArgs string
}

type OcRunner struct {
	CommandPath string
}

func runCommand(command string, commandPath string) (stdOut string, stdErr string, exitCode int) {
	commandArr := strings.Split(command, " ")
	path, _ := filepath.Abs(commandPath)
	cmd := exec.Command(path, commandArr...)

	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	err := cmd.Run()
	stdOut = outbuf.String()
	stdErr = errbuf.String()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			if stdErr == "" {
				stdErr = err.Error()
			}
			exitCode = 1 // unable to get error code
		}
	} else {
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}

	return
}

func (m *MinishiftRunner) RunCommand(command string) (stdOut string, stdErr string, exitCode int) {
	stdOut, stdErr, exitCode = runCommand(command, m.CommandPath)
	return
}

func (m *MinishiftRunner) Start() {
	m.RunCommand(fmt.Sprintf("start %s", m.CommandArgs))
}

func (m *MinishiftRunner) CDKSetup() {
	if (os.Getenv("MINISHIFT_USERNAME") == "") || (os.Getenv("MINISHIFT_PASSWORD") == "") {
		fmt.Println("Either MINISHIFT_USERNAME or MINISHIFT_PASSWORD is not set as environment variable")
		os.Exit(1)
	}
	m.RunCommand(fmt.Sprintf("setup-cdk --force --minishift-home %s", os.Getenv(constants.MiniShiftHomeEnv)))
}

func (m *MinishiftRunner) IsCDK() bool {
	cmdOut, _, _ := m.RunCommand("-h")
	return strings.Contains(cmdOut, "setup-cdk")
}

func (m *MinishiftRunner) EnsureRunning() {
	if m.GetStatus() != "Running" {
		m.Start()
	}
	m.CheckStatus("Running")
}

func (m *MinishiftRunner) IsRunning() bool {
	return m.GetStatus() == "Running"
}

func (m *MinishiftRunner) GetOcRunner() *OcRunner {
	if m.IsRunning() {
		return NewOcRunner()
	}
	return nil
}

func (m *MinishiftRunner) EnsureDeleted() {
	m.RunCommand("delete")
	m.CheckStatus("Does Not Exist")
}

func (m *MinishiftRunner) SetEnvFromEnvCmdOutput(dockerEnvVars string) error {
	lines := strings.Split(dockerEnvVars, "\n")
	var envKey, envVal string
	seenEnvVar := false
	for _, line := range lines {
		fmt.Println(line)
		if strings.HasPrefix("export ", line) {
			line = strings.TrimPrefix(line, "export ")
		}
		if _, err := fmt.Sscanf(line, "export %s=\"%s\"", &envKey, &envVal); err != nil {
			seenEnvVar = true
			fmt.Println(fmt.Sprintf("%s=%s", envKey, envVal))
			os.Setenv(envKey, envVal)
		}
	}
	if seenEnvVar == false {
		return fmt.Errorf("Error: No environment variables were found in docker-env command output: %s", dockerEnvVars)
	}
	return nil
}

func (m *MinishiftRunner) GetStatus() string {
	cmdOut, _, _ := m.RunCommand("status")
	return strings.Trim(cmdOut, " \n")
}

func (m *MinishiftRunner) CheckStatus(desired string) bool {
	return m.GetStatus() == desired
}

func NewOcRunner() *OcRunner {
	jsonDataPath := filepath.Join(os.Getenv(constants.MiniShiftHomeEnv), "machines", constants.MachineName+".json")
	instanceState.InstanceConfig, _ = instanceState.NewInstanceConfig(jsonDataPath)
	p := instanceState.InstanceConfig.OcPath
	return &OcRunner{CommandPath: p}
}

func (k *OcRunner) RunCommand(command string) (stdOut string, stdErr string, exitCode int) {
	stdOut, stdErr, exitCode = runCommand(command, k.CommandPath)
	return
}
