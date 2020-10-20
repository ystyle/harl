package model

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

type Config struct {
	Build  *Build
	Reload *Reload
}

type Build struct {
	Project   string `yaml:"-"`
	BuildType string
	Excludes  []string
	Includes  []string
	Nfsdir    string
	Delay     int
}

type Reload struct {
	Dir string
	//Telnet      string
	Com         string
	BundleName  string
	AbilityName string
}

func (config Config) setDefault() {
	if config.Build.Delay <= 50 {
		config.Build.Delay = 100
	}
	if config.Build.Project == "" {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		config.Build.Project = wd
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
