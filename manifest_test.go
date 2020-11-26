package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	testManifest = `
version: 2
default:
  check:
    path: /hello
  resource:
    cpu: 200:2000
  vars:
    hello: world
    hello2: world
  build:
    - echo hello {{.Env.WORLD}}
  package:
    - FROM nginx
    - RUN echo {{.Vars.hello}} > /usr/share/nginx/html/index.html
    - RUN echo {{stringsToUpper .Vars.hello2}} > /usr/share/nginx/html/index2.html

dev:
  check:
    port: 8888
  resource:
    mem: 200:-
  vars:
    hello2: world2
  build:
    - echo hello2 {{.Env.WORLD}}
`
	testManifestBuild = `#!/bin/bash
set -eux
echo hello2`
	testManifestPackage = `FROM nginx
RUN echo world > /usr/share/nginx/html/index.html
RUN echo WORLD2 > /usr/share/nginx/html/index2.html`
)

func TestLoadManifestFile(t *testing.T) {
	var m Manifest
	var p Profile
	var err error
	err = LoadManifest([]byte(testManifest), &m)
	assert.NoError(t, err)
	p, err = m.Profile("dev")
	assert.NoError(t, err)
	assert.Equal(t, "dev", p.Profile)
	assert.Equal(t, 8888, p.Check.Port)
	assert.Equal(t, "/hello", p.Check.Path)
	assert.Equal(t, "200:-", p.Resource.MEM.String())
	assert.Equal(t, "200:2000", p.Resource.CPU.String())
	var buf []byte
	buf, err = p.GenerateBuild()
	assert.NoError(t, err)
	assert.Equal(t, []byte(testManifestBuild), bytes.TrimSpace(buf))
	buf, err = p.GeneratePackage()
	assert.NoError(t, err)
	assert.Equal(t, []byte(testManifestPackage), bytes.TrimSpace(buf))
}
