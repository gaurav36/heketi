//
// Copyright (c) 2014 The heketi Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package ssh

// :TODO: This file needs logging

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
)

type SshExec struct {
	clientConfig *ssh.ClientConfig
}

func getKeyFile(file string) (key ssh.Signer, err error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	key, err = ssh.ParsePrivateKey(buf)
	if err != nil {
		fmt.Print(err)
		return
	}
	return
}

func NewSshExecWithAuth(user string) *SshExec {

	sshexec := &SshExec{}

	authSocket := os.Getenv("SSH_AUTH_SOCK")
	if authSocket == "" {
		log.Fatal("SSH_AUTH_SOCK required, check that your ssh agent is running")
		return nil
	}

	agentUnixSock, err := net.Dial("unix", authSocket)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	agent := agent.NewClient(agentUnixSock)
	signers, err := agent.Signers()
	if err != nil {
		log.Fatal(err)
		return nil
	}

	sshexec.clientConfig = &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signers...)},
	}

	return sshexec
}

func NewSshExecWithKeyFile(user string, file string) *SshExec {

	var key ssh.Signer
	var err error

	sshexec := &SshExec{}
	// Now in the main function DO:
	if key, err = getKeyFile(file); err != nil {
		fmt.Println("Unable to get keyfile")
		return nil
	}
	// Define the Client Config as :
	sshexec.clientConfig = &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
	}

	return sshexec
}

// This function was taken from https://github.com/coreos/etcd-manager/blob/master/main.go
func (s *SshExec) ConnectAndExec(host string, commands []string, wg *sync.WaitGroup) ([]string, error) {
	if wg != nil {
		defer wg.Done()
	}

	buffers := make([]string, len(commands))

	// :TODO: Will need a timeout here in case the server does not respond
	client, err := ssh.Dial("tcp", host, s.clientConfig)
	if err != nil {
		//logStderr(host, fmt.Sprintf("failed to create SSH connection: %s\n", err))
		return nil, err
	}

	for index, command := range commands {

		session, err := client.NewSession()
		if err != nil {
			fmt.Printf("Unable to connect to [%v]: %v\n", command, err)
			//logStderr(host, fmt.Sprintf("failed to create SSH session: %s", err))
			return nil, err
		}

		defer session.Close()

		var b bytes.Buffer
		session.Stdout = &b

		// Save the buffer for the caller

		if err := session.Run(command); err != nil {
			fmt.Printf("Failed to run command [%v] on [%v]: %v\n", command, host, err)
			// logStderr(host, fmt.Sprintf("error running command: %s", err))
			return nil, err
		}
		buffers[index] = b.String()
	}

	return buffers, nil
}
