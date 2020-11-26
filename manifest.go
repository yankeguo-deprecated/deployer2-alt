package main

import (
	"errors"
	"github.com/imdario/mergo"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

const (
	ManifestVersion = 2
)

type Manifest struct {
	Version  int                `yaml:"version"`
	Default  Profile            `yaml:"default"`
	Profiles map[string]Profile `yaml:",inline"`
}

func LoadManifest(buf []byte, m *Manifest) (err error) {
	if err = yaml.UnmarshalStrict(buf, m); err != nil {
		return
	}
	if m.Version != ManifestVersion {
		err = errors.New("描述文件 deployer.yml 中缺少版本号 version: 2")
		return
	}
	return
}

func LoadManifestFile(file string, m *Manifest) (err error) {
	var buf []byte
	if buf, err = ioutil.ReadFile(file); err != nil {
		return
	}
	if err = LoadManifest(buf, m); err != nil {
		return
	}
	return
}

func (m Manifest) Profile(name string) (p Profile, err error) {
	p = m.Profiles[name]
	p.Profile = name
	if err = mergo.Merge(&p, m.Default); err != nil {
		return
	}
	return
}
