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

func (w *SignalWatcher) Start() chan *signal {
	ch := make(chan *signal)
	go func() {
		for {
			time.Sleep(w.interval)

			s, err := w.Once()
			if err != nil {
				log.Printf("ERROR: %v", err)
				continue
			}
			if s == nil {
				continue
			}

			log.Printf("INFO: a signal object is found: %+v", s)
			ch <- s
		}
	}()

	return ch
}

func (w *SignalWatcher) Once() (*signal, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String(w.key),
	}

	log.Printf("DEBUG: checking a signal object at s3://%s/%s", w.bucket, w.key)
	output, err := w.s3.GetObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			log.Println("DEBUG: a signal object is not found")
			return nil, nil
		}
		return nil, err
	}

	data, err := ioutil.ReadAll(output.Body)
	if err != nil {
		return nil, err
	}

	s := signal{}
	err = json.Unmarshal(data, &s)
	if err != nil {
		return nil, err
	}

	return &s, nil
}
