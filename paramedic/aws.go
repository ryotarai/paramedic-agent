package paramedic

import (
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3 interface {
	GetObject(*s3.GetObjectInput) (*s3.GetObjectOutput, error)
}

type CloudWatchLogs interface {
	DescribeLogStreams(*cloudwatchlogs.DescribeLogStreamsInput) (*cloudwatchlogs.DescribeLogStreamsOutput, error)
	PutLogEvents(*cloudwatchlogs.PutLogEventsInput) (*cloudwatchlogs.PutLogEventsOutput, error)
}
