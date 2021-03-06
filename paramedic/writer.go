package paramedic

import (
	"errors"
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
	partialStr    string
	closed        bool

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
		closed:   false,

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
		closed := false
		for {
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
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.closed {
		return 0, errors.New("already closed")
	}

	text := w.partialStr + string(p)
	lines := strings.Split(text, "\n")
	w.partialStr = lines[len(lines)-1]
	lines = lines[:len(lines)-1]
	for _, l := range lines {
		e := logEntry{
			text:      l,
			timestamp: time.Now(),
		}
		w.buffer = append(w.buffer, e)
	}

	return len(p), nil
}

func (w *CloudWatchLogsWriter) Close() error {
	log.Println("[DEBUG] Closing CloudWatchLogsWriter")
	w.closed = true
	w.flushPartialStr()
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

func (w *CloudWatchLogsWriter) flushPartialStr() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if len(w.partialStr) == 0 {
		return
	}

	e := logEntry{
		text:      w.partialStr,
		timestamp: time.Now(),
	}
	w.buffer = append(w.buffer, e)
}

func (w *CloudWatchLogsWriter) flushBuffer() {
	log.Println("[DEBUG] Flushing log buffer")
	for {
		remaining := w.flushBufferOnce()
		if remaining == 0 {
			break
		}
	}
}

func (w *CloudWatchLogsWriter) flushBufferOnce() int {
	w.mutex.Lock()
	batch := []logEntry{}
	batchSize := 0
	for _, e := range w.buffer {
		s := len(e.text) + 26      // 26 is size of header of log event
		if batchSize+s > 1048576 { // 1048576 is max size of a single batch
			break
		}
		if len(batch) >= 10000 { // 10000 is max count of records in a single batch
			break
		}
		batch = append(batch, e)
		batchSize += s
	}
	w.buffer = w.buffer[len(batch):]
	remaining := len(w.buffer)
	w.mutex.Unlock()

	if len(batch) == 0 {
		return 0
	}

	sleep := time.Second * 1
	for {
		err := w.putEvents(batch)
		if err == nil {
			break
		}

		log.Printf("[WARN] Uploading logs failed: %s", err)
		log.Printf("[WARN] will retry after %s", sleep.String())
		time.Sleep(sleep)
		sleep *= 2
	}

	return remaining
}

func (w *CloudWatchLogsWriter) putEvents(entries []logEntry) error {
	log.Printf("[DEBUG] Uploading %d log entries", len(entries))

	events := []*cloudwatchlogs.InputLogEvent{}
	for _, e := range entries {
		event := &cloudwatchlogs.InputLogEvent{
			Message:   aws.String(e.text),
			Timestamp: aws.Int64(e.timestamp.UnixNano() / 1000 / 1000),
		}
		events = append(events, event)
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
