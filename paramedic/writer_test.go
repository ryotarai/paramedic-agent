package paramedic

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/golang/mock/gomock"
	"github.com/ryotarai/paramedic-agent/mock"
)

func TestCloudWatchLogsWriter_Start(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	cwlogs := mock.NewMockCloudWatchLogs(mockCtrl)
	w := NewCloudWatchLogsWriter(cwlogs, "g", "s", time.Hour)

	output := &cloudwatchlogs.PutLogEventsOutput{
		NextSequenceToken: aws.String("dummy"),
	}
	//=============================================
	w.Write([]byte("abc\ndef"))
	cwlogs.EXPECT().PutLogEvents(gomock.Any()).Do(func(input *cloudwatchlogs.PutLogEventsInput) {
		got := *input.LogEvents[0].Message
		expect := "abc"
		if got != expect {
			t.Errorf("got %s but expected %s", got, expect)
		}
	}).Return(output, nil)
	w.flushBuffer()
	//=============================================
	w.Write([]byte("ghi\njkl"))
	cwlogs.EXPECT().PutLogEvents(gomock.Any()).Do(func(input *cloudwatchlogs.PutLogEventsInput) {
		got := *input.LogEvents[0].Message
		expect := "defghi"
		if got != expect {
			t.Errorf("got %s but expected %s", got, expect)
		}
	}).Return(output, nil)
	w.flushBuffer()
	//=============================================
	cwlogs.EXPECT().PutLogEvents(gomock.Any()).Do(func(input *cloudwatchlogs.PutLogEventsInput) {
		got := *input.LogEvents[0].Message
		expect := "jkl"
		if got != expect {
			t.Errorf("got %s but expected %s", got, expect)
		}
	}).Return(output, nil)
	w.flushPartialStr()
}
