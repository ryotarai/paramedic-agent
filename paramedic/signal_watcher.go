package paramedic

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

type SignalWatcher struct {
	bucket   string
	key      string
	interval time.Duration
	s3       S3
}

type signal struct {
	Signal int `json:"signal"` // signal sent to the process
}

func (w *SignalWatcher) Start() chan signal {
	ch := make(chan signal)
	go func() {
		input := &s3.GetObjectInput{
			Bucket: aws.String(w.bucket),
			Key:    aws.String(w.key),
		}
		for {
			time.Sleep(w.interval)

			output, err := w.s3.GetObject(input)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
					continue
				}
				log.Printf("ERROR: %v", err)
				continue
			}

			data, err := ioutil.ReadAll(output.Body)
			if err != nil {
				log.Printf("ERROR: %v", err)
				continue
			}

			s := signal{}
			err = json.Unmarshal(data, &s)
			if err != nil {
				log.Printf("ERROR: %v", err)
				continue
			}

			ch <- s
		}
	}()

	return ch
}
