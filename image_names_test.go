package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestImageNames(t *testing.T) {
	imageNames := ImageNames{"a", "b"}
	assert.Equal(t, "a", imageNames.Primary())
	remoteImageNames := imageNames.Derive("hello")
	assert.Equal(t, ImageNames{"hello/a", "hello/b"}, remoteImageNames)
}
