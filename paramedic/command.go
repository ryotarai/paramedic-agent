package paramedic

import (
	"io"
	"log"
	"os/exec"
)

type Command struct {
	Name   string
	Args   []string
	Writer io.Writer
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
	cmd := exec.Command(c.Name, c.Args...)
	cmd.Stdout = c.Writer
	cmd.Stderr = c.Writer

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	ch := make(chan error)
	go func() {
		ch <- cmd.Wait()
	}()
	return ch, nil
}
