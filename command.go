package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
)

type Command struct {
	cmd    *exec.Cmd
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	env    map[string]string
}

func NewCommand(command string, args ...string) *Command {
	c := &Command{}
	c.cmd = exec.Command(command, args...)
	c.stdout = bytes.NewBuffer([]byte{})
	c.stderr = bytes.NewBuffer([]byte{})
	c.cmd.Stdout = c.stdout
	c.cmd.Stderr = c.stderr
	c.env = make(map[string]string)
	return c
}

func NewGoCommand(args ...string) *Command {
	return NewCommand("go", args...)
}

func NewGitCommand(args ...string) *Command {
	return NewCommand("git", args...)
}

func (c *Command) CleanArgs() {
	c.cmd.Args = []string{}
}

func (c *Command) SetEnv(name, value string) *Command {
	c.env[name] = value
	return c
}

func (c *Command) AppendArgs(arg ...string) *Command {
	c.cmd.Args = append(c.cmd.Args, arg...)
	return c
}

func (c *Command) Start() error {
	c.cmd.Env = os.Environ()
	for k, v := range c.env {
		c.cmd.Env = append(c.cmd.Env, k+"="+v)
	}
	return c.cmd.Start()
}

func (c *Command) Wait() error {
	err := c.cmd.Wait()
	if err != nil {
		strStderr := string(c.Stderr())
		if strStderr == "" {
			strStderr = string(c.Stdout())
		}
		return errors.New(err.Error() + "\n" + strings.TrimSpace(strStderr))
	}

	return nil
}

func (c *Command) Stdout() []byte {
	return c.stdout.Bytes()
}

func (c *Command) Stderr() []byte {
	return c.stderr.Bytes()
}

func (c *Command) JSONStdout(i any) error {
	return json.Unmarshal(c.stdout.Bytes(), i)
}
