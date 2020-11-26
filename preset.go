package main

import (
	"encoding/json"
	"errors"
	"github.com/guoyk93/tempfile"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type Preset struct {
	Registry         string                 `yaml:"registry"`
	Annotations      map[string]string      `yaml:"annotations"`
	ImagePullSecrets []string               `yaml:"imagePullSecrets"`
	Resource         UniversalResourceList  `yaml:"resource"`
	Kubeconfig       map[string]interface{} `yaml:"kubeconfig"`
	Dockerconfig     struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	} `yaml:"dockerconfig"`
}

func LoadPresetFromHome(cluster string, p *Preset) (err error) {
	var home string
	if home = os.Getenv("HOME"); len(home) == 0 {
		err = errors.New("缺少环境变量 $HOME")
		return
	}
	filename := filepath.Join(home, ".deployer2", "preset-"+cluster+".yml")
	log.Printf("加载集群配置: %s", filename)
	if err = LoadPresetFile(filename, p); err != nil {
		return
	}
	return
}

func LoadPresetFile(filename string, p *Preset) (err error) {
	var buf []byte
	if buf, err = ioutil.ReadFile(filename); err != nil {
		return
	}
	if err = LoadPreset(buf, p); err != nil {
		return
	}
	return
}

func LoadPreset(buf []byte, p *Preset) error {
	return yaml.Unmarshal(buf, p)
}

func (p Preset) GenerateKubeconfig() []byte {
	if p.Kubeconfig == nil {
		return []byte{}
	}
	buf, err := yaml.Marshal(p.Kubeconfig)
	if err != nil {
		panic(err)
	}
	return buf
}

func (p Preset) GenerateDockerconfig() []byte {
	buf, err := json.Marshal(p.Dockerconfig)
	if err != nil {
		panic(err)
	}
	return buf
}

func (p Preset) GenerateFiles() (dcDir string, kcFile string, err error) {
	var dcFile string
	if dcDir, dcFile, err = tempfile.WriteDirFile(
		p.GenerateDockerconfig(),
		"deployer-dockerconfig",
		"config.json",
		false,
	); err != nil {
		return
	}
	log.Printf("生成 Docker 配置文件: %s", dcFile)
	if kcFile, err = tempfile.WriteFile(
		p.GenerateKubeconfig(),
		"deployer-kubeconfig",
		".yml",
		false,
	); err != nil {
		return
	}
	log.Printf("生成 Kubeconfig 文件: %s", kcFile)
	return
}
