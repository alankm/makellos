package core

import (
	"io/ioutil"

	"github.com/sisatech/raft"
	"gopkg.in/yaml.v2"
)

type config struct {
	Version   string      `yaml:"version"`
	Bind      string      `yaml:"bind"`
	Advertise string      `yaml:"advertise"`
	Base      string      `yaml:"base"`
	Raft      raft.Config `yaml:"raft"`
	// Modules
}

func (c *config) load(path string) error {
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
