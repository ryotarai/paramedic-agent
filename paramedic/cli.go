package paramedic

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
)

type CLI struct {
}

func NewCLI() *CLI {
	return &CLI{}
}

type Options struct {
	Args           []string
	S3Bucket       string
	S3KeyPrefix    string
	UploadInterval time.Duration
	MaxChunkSize   int
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
	fs.StringVar(&options.S3Bucket, "s3-bucket", "", "Output S3 bucket")
	fs.StringVar(&options.S3KeyPrefix, "s3-key-prefix", "", "Output S3 key prefix")
	fs.IntVar(&options.MaxChunkSize, "max-chunk-size", 1024*1024, "Max size of chunks of output buffer (in byte)")
	intervalStr := fs.String("upload-interval", "30s", "Interval to upload output data")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	if options.S3Bucket == "" {
		return nil, errors.New("-s3-bucket is mandatory option")
	}
	if options.S3KeyPrefix == "" {
		return nil, errors.New("-s3-key-prefix is mandatory option")
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
	writer, err := NewS3Writer(options.S3Bucket, options.S3KeyPrefix, options.UploadInterval, options.MaxChunkSize)
	if err != nil {
		return err
	}

	writer.StartUploading()
	defer writer.Finalize()

	cmd := NewCommand(options.Args[0], options.Args[1:], writer)
	return cmd.Run()
}
