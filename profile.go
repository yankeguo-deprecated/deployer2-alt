package main

import (
	"bytes"
	"github.com/acicn/deployer2/pkg/tmplfuncs"
	"github.com/guoyk93/tempfile"
	"log"
	"os"
	"strings"
	"text/template"
)

type ProfileBuilder struct {
	Image      string   `yaml:"image"`
	CacheGroup string   `yaml:"cacheGroup"`
	Caches     []string `yaml:"caches"`
}

type Profile struct {
	Profile  string                 `yaml:"-"`
	Resource UniversalResourceList  `yaml:"resource"`
	Check    UniversalCheck         `yaml:"check"`
	Build    []string               `yaml:"build"`
	Builder  ProfileBuilder         `yaml:"builder"`
	Package  []string               `yaml:"package"`
	Vars     map[string]interface{} `yaml:"vars"`
}

func (p *Profile) Render(src string) (out []byte, err error) {
	var tmpl *template.Template
	if tmpl, err = template.New("").
		Option("missingkey=zero").
		Funcs(tmplfuncs.Funcs).Parse(src); err != nil {
		return
	}

	envs := map[string]string{}
	for _, env := range os.Environ() {
		splits := strings.SplitN(env, "=", 2)
		if len(splits) == 2 {
			envs[splits[0]] = splits[1]
		}
	}
	data := map[string]interface{}{
		"Env":     envs,
		"Vars":    p.Vars,
		"Profile": p.Profile,
	}

	buf := &bytes.Buffer{}
	if err = tmpl.Execute(buf, data); err != nil {
		return
	}
	out = buf.Bytes()
	return
}

func (p *Profile) GenerateBuild() ([]byte, error) {
	s := &strings.Builder{}
	s.WriteString("#!/bin/bash\nset -eux\n")
	for _, l := range p.Build {
		s.WriteString(l)
		s.WriteRune('\n')
	}
	return p.Render(s.String())
}

func (p *Profile) GeneratePackage() ([]byte, error) {
	return p.Render(strings.Join(p.Package, "\n"))
}

func (p *Profile) PrintGeneratedContent(name string, content string) {
	sb := &strings.Builder{}
	sb.WriteRune('\n')
	sb.WriteString(name)
	sb.WriteString(":\n--------------------------------------------------\n")
	sb.WriteString(strings.TrimSpace(content))
	sb.WriteString("\n--------------------------------------------------")
	log.Println(sb.String())
	if strings.Contains(content, "<no value>") {
		log.Println("警告：检查到渲染结果出现 <no value>，请确认：")
		log.Printf("  1. 环境名 %s 是否正确 (环境名可能取自 Jenkins 任务后缀)", p.Profile)
		log.Printf("  2. 环境 %s 的 vars 字段是否缺失某些变量", p.Profile)
	}
}

func (p *Profile) GenerateFiles() (buildFile string, packageFile string, err error) {
	var buf []byte
	if buf, err = p.GenerateBuild(); err != nil {
		return
	}
	p.PrintGeneratedContent("构建脚本", string(buf))
	if buildFile, err = tempfile.WriteFile(buf, "deployer-build", ".sh", true); err != nil {
		return
	}
	if buf, err = p.GeneratePackage(); err != nil {
		return
	}
	p.PrintGeneratedContent("打包脚本", string(buf))
	if packageFile, err = tempfile.WriteFile(buf, "deployer-package", ".dockerfile", false); err != nil {
		return
	}
	return
}
