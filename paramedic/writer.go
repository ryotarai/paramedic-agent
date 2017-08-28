package paramedic

import (
	"bufio"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

type logEntry struct {
	text      string
	timestamp time.Time
}

type CloudWatchLogsWriter struct {
	client        CloudWatchLogs
	group         string
	stream        string
	interval      time.Duration
	buffer        []logEntry
	mutex         sync.Mutex
	sequenceToken string

	closeCh chan struct{}
	doneCh  chan struct{}
}

func NewCloudWatchLogsWriter(client CloudWatchLogs, group string, stream string, interval time.Duration) *CloudWatchLogsWriter {
	return &CloudWatchLogsWriter{
		client:   client,
		group:    group,
		stream:   stream,
		interval: interval,
		buffer:   []logEntry{},
		mutex:    sync.Mutex{},

		closeCh: make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
}

func (w *CloudWatchLogsWriter) Start() error {
	err := w.createStream()
	if err != nil {
		return err
	}

	go func() {
		for {
			closed := false
			select {
			case <-w.closeCh:
				closed = true
			case <-time.After(w.interval):
			}
			w.flushBuffer()
			if closed {
				w.doneCh <- struct{}{}
				break
			}
		}
	}()

	return nil
}

func (w *CloudWatchLogsWriter) Write(p []byte) (int, error) {
	log.Printf("%s", string(p))

	w.mutex.Lock()
	defer w.mutex.Unlock()

	e := logEntry{
		text:      string(p),
		timestamp: time.Now(),
	}
	w.buffer = append(w.buffer, e)

	return len(p), nil
}

func (w *CloudWatchLogsWriter) Close() error {
	log.Println("DEBUG: closing CloudWatchLogsWriter")
	w.closeCh <- struct{}{}
	<-w.doneCh
	return nil
}

func (w *CloudWatchLogsWriter) createStream() error {
	input := &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(w.group),
		LogStreamName: aws.String(w.stream),
	}
	_, err := w.client.CreateLogStream(input)
	if err != nil {
		return err
	}

	return nil
}

func (w *CloudWatchLogsWriter) flushBuffer() {
	log.Println("DEBUG: flushing log buffer")

	w.mutex.Lock()
	b := w.buffer
	w.buffer = []logEntry{}
	w.mutex.Unlock()

	sleep := time.Second * 1
	for {
		err := w.putEvents(b)
		if err == nil {
			break
		}

		log.Printf("WARN: uploading logs failed. will retry after %s", sleep.String())
		time.Sleep(sleep)
		sleep *= 2
	}
}

func (w *CloudWatchLogsWriter) putEvents(entries []logEntry) error {
	log.Printf("DEBUG: uploading %d log entries", len(entries))
	// TODO: batch size

	events := []*cloudwatchlogs.InputLogEvent{}
	for _, e := range entries {
		reader := strings.NewReader(e.text)
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			event := &cloudwatchlogs.InputLogEvent{
				Message:   aws.String(scanner.Text()),
				Timestamp: aws.Int64(e.timestamp.UnixNano() / 1000 / 1000),
			}
			events = append(events, event)
		}
	}

	input := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(w.group),
		LogStreamName: aws.String(w.stream),
		LogEvents:     events,
	}
	if w.sequenceToken != "" {
		input.SequenceToken = aws.String(w.sequenceToken)
	}

	output, err := w.client.PutLogEvents(input)
	if err != nil {
		return err
	}

	w.sequenceToken = *output.NextSequenceToken
	return nil
}
