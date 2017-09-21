package paramedic

import (
	"io/ioutil"
	"net/http"
	"os"
)

const instanceIDURL = "http://169.254.169.254/2016-09-02/meta-data/instance-id"

func fetchInstanceID() (string, error) {
	if id := os.Getenv("AWS_SSM_INSTANCE_ID"); id != "" {
		return id, nil
	}

	resp, err := http.Get(instanceIDURL)
	if err != nil {
		return "", err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	resp.Body.Close()

	return string(b), nil
}
