package paramedic

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Command struct {
	s3     S3
	bucket string
	key    string
	writer io.Writer

	cmd *exec.Cmd
}

func NewCommand(s3 S3, bucket string, key string, writer io.Writer) *Command {
	return &Command{
		s3:     s3,
		bucket: bucket,
		key:    key,
		writer: writer,
	}
}

func (c *Command) Start() (chan error, error) {
	f, err := ioutil.TempFile("", "paramedic")
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(f.Name(), 0700); err != nil {
		return nil, err
	}
	if err := c.download(f); err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	c.cmd = exec.Command(f.Name())
	c.cmd.Stdout = c.writer
	c.cmd.Stderr = c.writer

	log.Printf("INFO: starting %s", f.Name())
	if err := c.cmd.Start(); err != nil {
		return nil, err
	}

	ch := make(chan error)
	go func() {
		ch <- c.cmd.Wait()
		os.Remove(f.Name())
	}()
	return ch, nil
}

func (c *Command) Signal(sig os.Signal) error {
	log.Printf("INFO: signal %d is sent to pid %d", sig, c.cmd.Process.Pid)
	return c.cmd.Process.Signal(sig)
}

func (c *Command) download(f *os.File) error {
	log.Printf("INFO: downloading a script from s3://%s/%s to %s", c.bucket, c.key, f.Name())

	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(c.key),
	}
	output, err := c.s3.GetObject(input)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, output.Body)
	if err != nil {
		return err
	}
	output.Body.Close()

	return nil
}
