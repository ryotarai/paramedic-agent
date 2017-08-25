package paramedic

import (
	"io"
	"log"
	"os"
	"os/exec"
)

type Command struct {
	Name   string
	Args   []string
	Writer io.Writer

	cmd *exec.Cmd
}

func NewCommand(name string, args []string, writer io.Writer) *Command {
	return &Command{
		Name:   name,
		Args:   args,
		Writer: writer,
	}
}

func (c *Command) Start() (chan error, error) {
	log.Printf("running %s %#v", c.Name, c.Args)
	c.cmd = exec.Command(c.Name, c.Args...)
	c.cmd.Stdout = c.Writer
	c.cmd.Stderr = c.Writer

	err := c.cmd.Start()
	if err != nil {
		return nil, err
	}

	ch := make(chan error)
	go func() {
		ch <- c.cmd.Wait()
	}()
	return ch, nil
}

func (c *Command) Signal(sig os.Signal) error {
	log.Printf("INFO: signal %d is sent to pid %d", sig, c.cmd.Process.Pid)
	return c.cmd.Process.Signal(sig)
}
