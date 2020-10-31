package model

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

type Config struct {
	Watch   *Watch
	Nfs     *Nfs
	Shell   *Shell
	Command map[string][]string `yaml:"command,omitempty"`
}

type Watch struct {
	Project  string `yaml:"-"`
	Excludes []string
	Includes []string
	Delay    int
}

type Nfs struct {
	Ldir string
	Rdir string
}

type Shell struct {
	Com string
}

type Build struct {
	Nfsdir string
}

func (config Config) setDefault() {
	if config.Watch.Delay <= 50 {
		config.Watch.Delay = 100
	}
	if config.Watch.Project == "" {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		config.Watch.Project = wd
	}
	if config.Command == nil {
		config.Command = make(map[string][]string)
	}
}

func ReadConfig() Config {
	bs, err := ioutil.ReadFile(".harl.yaml")
	if err != nil {
		panic(err)
	}
	var config Config
	err = yaml.Unmarshal(bs, &config)
	if err != nil {
		panic(err)
	}
	config.setDefault()
	return config
}
