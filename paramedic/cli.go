package paramedic

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const agentExitCode = 254

type CLI struct {
}

func NewCLI() *CLI {
	return &CLI{}
}

type Options struct {
	Args            []string
	OutputLogGroup  string
	OutputLogStream string
	SignalS3Bucket  string
	SignalS3Key     string
	ScriptS3Bucket  string
	ScriptS3Key     string
	UploadInterval  time.Duration
	SignalInterval  time.Duration
}

func (c *CLI) Start() int {
	options, err := c.parseFlag(os.Args[0], os.Args[1:])
	if err != nil {
		log.Println(err)
		return 1
	}

	err, code := c.startWithOptions(options)
	if err != nil {
		log.Println(err)
		return code
	}

	return 0
}

func (c *CLI) parseFlag(name string, args []string) (*Options, error) {
	options := &Options{}

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.StringVar(&options.OutputLogGroup, "output-log-group", "", "Output log group")
	fs.StringVar(&options.OutputLogStream, "output-log-stream", "", "Output log stream")
	fs.StringVar(&options.SignalS3Bucket, "signal-s3-bucket", "", "Signal S3 bucket")
	fs.StringVar(&options.SignalS3Key, "signal-s3-key", "", "Signal S3 key")
	fs.StringVar(&options.ScriptS3Bucket, "script-s3-bucket", "", "Script S3 bucket")
	fs.StringVar(&options.ScriptS3Key, "script-s3-key", "", "Script S3 key")
	uploadIntervalStr := fs.String("upload-interval", "10s", "Interval to upload output")
	signalIntervalStr := fs.String("signal-interval", "10s", "Interval to check signal")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	if options.OutputLogGroup == "" {
		return nil, errors.New("-output-log-group is mandatory option")
	}
	if options.OutputLogStream == "" {
		return nil, errors.New("-output-log-stream is mandatory option")
	}
	if options.SignalS3Bucket == "" {
		return nil, errors.New("-signal-s3-bucket is mandatory option")
	}
	if options.SignalS3Key == "" {
		return nil, errors.New("-signal-s3-key is mandatory option")
	}
	if options.ScriptS3Bucket == "" {
		return nil, errors.New("-script-s3-bucket is mandatory option")
	}
	if options.ScriptS3Key == "" {
		return nil, errors.New("-script-s3-key is mandatory option")
	}

	options.Args = fs.Args()
	if len(options.Args) < 1 {
		return nil, errors.New("command is not specified")
	}

	d, err := time.ParseDuration(*uploadIntervalStr)
	if err != nil {
		return nil, err
	}
	options.UploadInterval = d

	d, err = time.ParseDuration(*signalIntervalStr)
	if err != nil {
		return nil, err
	}
	options.SignalInterval = d

	return options, nil
}

func (c *CLI) startWithOptions(options *Options) (error, int) {
	sess := session.Must(session.NewSession())
	s3 := s3.New(sess)
	// cwlogs := cloudwatchlogs.New(sess)

	cmd := NewCommand(options.Args[0], options.Args[1:], os.Stdout)
	cmdCh, err := cmd.Start()
	if err != nil {
		return err, agentExitCode
	}

	watcher := SignalWatcher{
		s3:       s3,
		bucket:   options.SignalS3Bucket,
		key:      options.SignalS3Key,
		interval: options.SignalInterval,
	}
	signalCh := watcher.Start()

L:
	for {
		select {
		case err := <-cmdCh:
			// command exited
			if err != nil {
				if eErr, ok := err.(*exec.ExitError); ok {
					if s, ok := eErr.Sys().(syscall.WaitStatus); ok {
						exitStatus := s.ExitStatus()
						return fmt.Errorf("command exited with %d", exitStatus), exitStatus
					}
					return errors.New("error does not implement syscall.WaitStatus"), agentExitCode
				}
				return err, agentExitCode
			}
			break L
		case signal := <-signalCh:
			// send signal
			cmd.Signal(syscall.Signal(signal.Signal))
		}
	}

	return nil, 0
}
