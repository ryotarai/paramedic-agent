package paramedic

import (
	"os/exec"

	"syscall"

	"bytes"

	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type CmdResultS3Uploader struct {
	Bucket string
	Key    string

	s3 *s3.S3
}

type CmdResult struct {
	ExitStatus int
	Error      string
}

func NewCmdResultS3Uploader(bucket string, key string) (*CmdResultS3Uploader, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	svc := s3.New(sess)

	return &CmdResultS3Uploader{
		Bucket: bucket,
		Key:    key,
		s3:     svc,
	}, nil
}

func (w *CmdResultS3Uploader) Upload(err error) error {
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return w.upload(err, -1)
	}

	ws, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		return w.upload(err, -1)
	}

	return w.upload(err, ws.ExitStatus())
}

func (w *CmdResultS3Uploader) upload(err error, exitStatus int) error {
	b, err := json.Marshal(CmdResult{
		Error:      err.Error(),
		ExitStatus: exitStatus,
	})
	if err != nil {
		return err
	}

	reader := bytes.NewReader(b)
	_, err = w.s3.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(w.Bucket),
		Key:    aws.String(w.Key),
		Body:   reader,
	})
	if err != nil {
		return err
	}
	return nil
}
