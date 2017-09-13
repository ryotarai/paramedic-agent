package paramedic

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	AWSCredentialProvider string `yaml:"AWSCredentialProvider"` // one of "" and "EC2Role"
}

func LoadConfig(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	err = yaml.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
