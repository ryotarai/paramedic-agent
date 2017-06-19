package paramedic

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
)

type Chunk struct {
	body   []byte
	index  int
	closed bool
}

func NewChunk(index int) *Chunk {
	return &Chunk{
		body:   []byte{},
		index:  index,
		closed: false,
	}
}

type S3Writer struct {
	MaxChunkSize int
	Interval     time.Duration
	Bucket       string
	KeyPrefix    string

	closed      bool
	buffer      []*Chunk
	s3          *s3.S3
	finalizeCh  chan struct{}
	finalizedCh chan struct{}
	mutex       sync.Mutex
}

func NewS3Writer(bucket string, key string, interval time.Duration, maxSize int) (*S3Writer, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	svc := s3.New(sess)

	c := NewChunk(1)
	return &S3Writer{
		MaxChunkSize: maxSize,
		Interval:     interval,
		Bucket:       bucket,
		KeyPrefix:    key,
		buffer:       []*Chunk{c},
		s3:           svc,
		finalizeCh:   make(chan struct{}),
		finalizedCh:  make(chan struct{}),
		mutex:        sync.Mutex{},
	}, nil
}

func (w *S3Writer) Close() {
	w.closed = true
	for _, chunk := range w.buffer {
		chunk.closed = true
	}

	w.finalizeCh <- struct{}{}
	<-w.finalizedCh
}

func (w *S3Writer) StartUploading() {
	go func() {
		for {
			fin := false

			after := time.After(w.Interval)
			select {
			case <-after:
			case <-w.finalizeCh:
				fin = true
			}

			w.uploadBuffer()

			if fin {
				w.finalizedCh <- struct{}{}
				return
			}
		}
	}()
}

func (w *S3Writer) uploadBuffer() {
	purgeCount := 0

	for _, chunk := range w.buffer {
		closed := chunk.closed
		reader := bytes.NewReader(chunk.body)
		key := fmt.Sprintf("%s%d.log", w.KeyPrefix, chunk.index)

		log.Printf("uploading to %s/%s", w.Bucket, key)

		_, err := w.s3.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(w.Bucket),
			Key:    aws.String(key),
			Body:   reader,
		})

		if err != nil {
			log.Printf("error in PutObject: %s", err)
			break
		}

		if closed {
			purgeCount++
		}
	}

	if purgeCount > 0 {
		log.Printf("purging %d chunks", purgeCount)
		w.mutex.Lock()
		w.buffer = w.buffer[purgeCount:]
		w.mutex.Unlock()
	}
}

func (w *S3Writer) Write(p []byte) (int, error) {
	if w.closed {
		return 0, errors.New("already closed")
	}

	// TODO: performance
	w.mutex.Lock()
	chunk := w.buffer[len(w.buffer)-1]
	if len(chunk.body)+len(p) > w.MaxChunkSize {
		chunk.closed = true

		chunk := NewChunk(chunk.index + 1)
		w.buffer = append(w.buffer, chunk)
		log.Printf("writing to new chunk#%d", chunk.index)
	}
	chunk.body = append(chunk.body, p...)
	w.mutex.Unlock()

	return len(p), nil
}
