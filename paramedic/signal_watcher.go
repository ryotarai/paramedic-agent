package paramedic

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type SignalWatcher struct {
	Bucket string
	Key    string

	s3 *s3.S3
}

func NewSignalWatcher(bucket string, key string) (*SignalWatcher, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	svc := s3.New(sess)

	return &SignalWatcher{
		Bucket: bucket,
		Key:    key,
		s3:     svc,
	}, nil
}
