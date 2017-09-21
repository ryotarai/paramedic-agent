package paramedic

import (
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/golang/mock/gomock"
	"github.com/ryotarai/paramedic-agent/mock"
)

func TestSignalWatcherOnce(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	s3m := mock.NewMockS3(mockCtrl)

	body := &stringReadCloser{strings.NewReader(`{"signal": 15}`)}
	s3m.EXPECT().GetObject(gomock.Any()).Return(&s3.GetObjectOutput{
		Body: body,
	}, nil)

	w := &SignalWatcher{
		bucket:   "paramedic",
		key:      "signal/a.json",
		interval: 100 * time.Millisecond,
		s3:       s3m,
	}

	sig, err := w.Once()
	if err != nil {
		t.Error(err)
	}
	if sig.Signal != 15 {
		t.Errorf("signal is %d but expected %d", sig.Signal, 15)
	}
}
