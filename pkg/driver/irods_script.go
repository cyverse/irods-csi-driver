/*
Copyright 2020 CyVerse
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

package driver

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"k8s.io/klog"
)

// python scripts for irods
const (
	irodsLsBin    = "irods_ls.py"
	irodsMkdirBin = "irods_mkdir.py"
	irodsRmdirBin = "irods_rmdir.py"
)

// IRODSMkdir creates a new directory
func IRODSMkdir(conn *IRODSConnection, path string) error {
	connectionArgs := conn.GetHostArgs()
	stdinValues := conn.GetLoginInfoArgs()
	safePath := strings.TrimRight(path, "/")
	args := []string{safePath}

	_, err := executeScript(irodsMkdirBin, connectionArgs, args, stdinValues)
	return err
}

// IRODSRmdir deletes a directory
func IRODSRmdir(conn *IRODSConnection, path string) error {
	connectionArgs := conn.GetHostArgs()
	stdinValues := conn.GetLoginInfoArgs()
	safePath := strings.TrimRight(path, "/")
	args := []string{safePath}

	_, err := executeScript(irodsRmdirBin, connectionArgs, args, stdinValues)
	return err
}

// IRODSLs lists entries in a directory
func IRODSLs(conn *IRODSConnection, path string) ([]string, error) {
	connectionArgs := conn.GetHostArgs()
	stdinValues := conn.GetLoginInfoArgs()
	safePath := strings.TrimRight(path, "/")
	args := []string{safePath}

	output, err := executeScript(irodsLsBin, connectionArgs, args, stdinValues)
	if err != nil {
		return nil, err
	}

	outputString := string(output)
	outputStringArr := strings.Split(outputString, "\n")

	return outputStringArr, nil
}

// executeScript runs external iRODS Exec scripts
func executeScript(bin string, connectionArgs []string, extraArgs []string, stdinValues []string) ([]byte, error) {
	args := []string{}

	args = append(args, connectionArgs...)
	args = append(args, extraArgs...)

	klog.V(4).Infof("Executing iRODS Exec (%s) with arguments (%v)", bin, args)
	command := exec.Command(bin, args...)
	stdin, err := command.StdinPipe()
	if err != nil {
		klog.Errorf("Accessing stdin failed: %v\niRODS command: %s\n Arguments: %v\n", err, bin, args)
		return nil, fmt.Errorf("accessing stdin failed: %v\niRODS command: %s\nArguments: %v", err, bin, args)
	}

	for _, stdinValue := range stdinValues {
		io.WriteString(stdin, stdinValue)
		io.WriteString(stdin, "\n")
	}
	stdin.Close()

	output, err := command.CombinedOutput()
	if err != nil {
		klog.Errorf("iRODS Exec failed: %v\niRODS command: %s\nArguments: %s\nOutput: %s\n", err, bin, args, string(output))
		return nil, fmt.Errorf("iRODS Exec failed: %v\niRODS command: %s\nArguments: %s\nOutput: %s", err, bin, args, string(output))
	}
	return output, err
}
