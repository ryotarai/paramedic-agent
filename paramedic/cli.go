package paramedic

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
)

type CLI struct {
}

func NewCLI() *CLI {
	return &CLI{}
}

type Options struct {
	Args              []string
	OutputS3Bucket    string
	OutputS3KeyPrefix string
	SignalS3Bucket    string
	SignalS3Key       string
	UploadInterval    time.Duration
	MaxChunkSize      int
}

func (c *CLI) Start() int {
	options, err := c.parseFlag(os.Args[0], os.Args[1:])
	if err != nil {
		fmt.Println(err)
		return 1
	}

	err = c.startWithOptions(options)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	return 0
}

func (c *CLI) parseFlag(name string, args []string) (*Options, error) {
	options := &Options{}

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.StringVar(&options.OutputS3Bucket, "output-s3-bucket", "", "Output S3 bucket")
	fs.StringVar(&options.OutputS3KeyPrefix, "output-s3-key-prefix", "", "Output S3 key prefix")
	fs.StringVar(&options.SignalS3Bucket, "signal-s3-bucket", "", "Signal S3 bucket")
	fs.StringVar(&options.SignalS3Key, "signal-s3-key", "", "Signal S3 key")
	fs.IntVar(&options.MaxChunkSize, "max-chunk-size", 1024*1024, "Max size of chunks of output buffer (in byte)")
	intervalStr := fs.String("upload-interval", "30s", "Interval to upload output data")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	if options.OutputS3Bucket == "" {
		return nil, errors.New("-output-s3-bucket is mandatory option")
	}
	if options.OutputS3KeyPrefix == "" {
		return nil, errors.New("-output-s3-key-prefix is mandatory option")
	}
	if options.SignalS3Bucket == "" {
		return nil, errors.New("-signal-s3-bucket is mandatory option")
	}
	if options.SignalS3Key == "" {
		return nil, errors.New("-signal-s3-key is mandatory option")
	}

	options.Args = fs.Args()
	if len(options.Args) < 1 {
		return nil, errors.New("command is not specified")
	}

	d, err := time.ParseDuration(*intervalStr)
	if err != nil {
		return nil, err
	}
	options.UploadInterval = d

	return options, nil
}

func (c *CLI) startWithOptions(options *Options) error {
	writer, err := NewS3Writer(options.OutputS3Bucket, options.OutputS3KeyPrefix, options.UploadInterval, options.MaxChunkSize)
	if err != nil {
		return err
	}

	writer.StartUploading()
	defer writer.Close()

	watcher, err := NewSignalWatcher(options.SignalS3Bucket, options.SignalS3Key)
	if err != nil {
		return err
	}

	cmd := NewCommand(options.Args[0], options.Args[1:], writer)
	cmdCh, err := cmd.Start()
	if err != nil {
		return err
	}

	select {
	case err := <-cmdCh:
	}
}
