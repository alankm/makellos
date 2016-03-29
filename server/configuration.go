package server

import (
	"io/ioutil"

	"github.com/sisatech/raft"

	"gopkg.in/yaml.v2"
)

type configuration struct {
	Version   string                       `yaml:"version"`
	Bind      string                       `yaml:"bind"`
	Advertise string                       `yaml:"advertise"`
	Base      string                       `yaml:"base"`
	Database  string                       `yaml:"database"`
	Modules   map[string]map[string]string `yaml:"modules"`
	Raft      raft.Config                  `yaml:"raft"`
	Storage   storageConfiguration         `yaml:"storage"`
}

func (c *configuration) load(path string) error {
	src, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(src, c)
	if err != nil {
		return err
	}
	return nil
}
