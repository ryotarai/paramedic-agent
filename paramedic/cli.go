package paramedic

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/s3"
)

const agentExitCode = 255

type CLI struct {
}

func NewCLI() *CLI {
	return &CLI{}
}

type Options struct {
	ShowVersion           bool
	OutputLogGroup        string
	OutputLogStreamPrefix string
	SignalS3Bucket        string
	SignalS3Key           string
	ScriptS3Bucket        string
	ScriptS3Key           string
	AWSCredentialProvider string
	UploadInterval        time.Duration
	SignalInterval        time.Duration
}

func (c *CLI) Start() int {
	options, err := c.parseFlag(os.Args[0], os.Args[1:])
	if err != nil {
		log.Printf("[ERROR] %s", err)
		return 1
	}

	if options.ShowVersion {
		fmt.Printf("paramedic-agent v%s\n", Version)
		return 0
	}

	loadConfigToOptions("/etc/paramedic-agent/config.yaml", options)

	if err := c.validateOptions(options); err != nil {
		log.Printf("[ERROR] %s", err)
		return 1
	}

	err, code := c.startWithOptions(options)
	if err != nil {
		log.Printf("[ERROR] %s", err)
	}

	return code
}

func loadConfigToOptions(path string, options *Options) {
	cfg, err := LoadConfig(path)
	if err != nil {
		log.Printf("[WARN] Failed to load a config file: %s", err)
		return
	}

	if options.AWSCredentialProvider == "" {
		options.AWSCredentialProvider = cfg.AWSCredentialProvider
	}
}

func (c *CLI) validateOptions(options *Options) error {
	if options.OutputLogGroup == "" {
		return errors.New("-output-log-group is mandatory option")
	}
	if options.OutputLogStreamPrefix == "" {
		return errors.New("-output-log-stream-prefix is mandatory option")
	}
	if options.SignalS3Bucket == "" {
		return errors.New("-signal-s3-bucket is mandatory option")
	}
	if options.SignalS3Key == "" {
		return errors.New("-signal-s3-key is mandatory option")
	}
	if options.ScriptS3Bucket == "" {
		return errors.New("-script-s3-bucket is mandatory option")
	}
	if options.ScriptS3Key == "" {
		return errors.New("-script-s3-key is mandatory option")
	}
	return nil
}

func (c *CLI) parseFlag(name string, args []string) (*Options, error) {
	options := &Options{}

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.BoolVar(&options.ShowVersion, "version", false, "Show version")
	fs.StringVar(&options.OutputLogGroup, "output-log-group", os.Getenv("PARAMEDIC_OUTPUT_LOG_GROUP"), "Output log group")
	fs.StringVar(&options.OutputLogStreamPrefix, "output-log-stream-prefix", os.Getenv("PARAMEDIC_OUTPUT_LOG_STREAM_PREFIX"), "Output log stream prefix")
	fs.StringVar(&options.SignalS3Bucket, "signal-s3-bucket", os.Getenv("PARAMEDIC_SIGNAL_S3_BUCKET"), "Signal S3 bucket")
	fs.StringVar(&options.SignalS3Key, "signal-s3-key", os.Getenv("PARAMEDIC_SIGNAL_S3_KEY"), "Signal S3 key")
	fs.StringVar(&options.ScriptS3Bucket, "script-s3-bucket", os.Getenv("PARAMEDIC_SCRIPT_S3_BUCKET"), "Script S3 bucket")
	fs.StringVar(&options.ScriptS3Key, "script-s3-key", os.Getenv("PARAMEDIC_SCRIPT_S3_KEY"), "Script S3 key")
	fs.StringVar(&options.AWSCredentialProvider, "credential-provider", "", "Credential provider (one of 'EC2Role')")
	uploadIntervalStr := fs.String("upload-interval", "10s", "Interval to upload output")
	signalIntervalStr := fs.String("signal-interval", "10s", "Interval to check signal")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
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
	log.Printf("[INFO] Starting paramedic-agent v%s", Version)

	sess := session.Must(session.NewSession())
	if options.AWSCredentialProvider == "EC2Role" {
		creds := ec2rolecreds.NewCredentials(sess)
		sess.Config.Credentials = creds
	}
	if region := os.Getenv("AWS_SSM_REGION_NAME"); region != "" {
		sess.Config.Region = aws.String(region)
	}

	s3 := s3.New(sess)
	cwlogs := cloudwatchlogs.New(sess)

	watcher := SignalWatcher{
		s3:       s3,
		bucket:   options.SignalS3Bucket,
		key:      options.SignalS3Key,
		interval: options.SignalInterval,
	}
	sig, err := watcher.Once()
	if err != nil {
		return err, agentExitCode
	}
	if sig != nil {
		log.Printf("[INFO] Exiting because signal object is found before starting a command")
		return nil, 0
	}

	instanceID, err := fetchInstanceID()
	if err != nil {
		return err, agentExitCode
	}

	logStream := fmt.Sprintf("%s%s", options.OutputLogStreamPrefix, instanceID)
	writer := NewCloudWatchLogsWriter(cwlogs, options.OutputLogGroup, logStream, options.UploadInterval)
	if err := writer.Start(); err != nil {
		return err, agentExitCode
	}

	cmd := NewCommand(s3, options.ScriptS3Bucket, options.ScriptS3Key, writer)
	cmdCh, err := cmd.Start()
	if err != nil {
		return err, agentExitCode
	}

	signalCh := watcher.Start()

	var exitStatus int
	var exitErr error

L:
	for {
		select {
		case err := <-cmdCh:
			// command exited
			exitStatus, exitErr = exitStatusFromError(err)
			if exitErr == nil {
				log.Printf("[INFO] The command exited with status %d", exitStatus)
			}
			break L
		case signal := <-signalCh:
			// send signal
			cmd.Signal(syscall.Signal(signal.Signal))
		}
	}

	if exitErr == nil {
		fmt.Fprintf(writer, "[exit status: %d]\n", exitStatus)
	} else {
		fmt.Fprintf(writer, "[%s]\n", exitErr)
	}
	writer.Close()

	return exitErr, exitStatus
}
