package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMarshalLabels(t *testing.T) {
	type testStruct struct {
		Hello bool `json:"hello"`
		World bool `json:"world"`
		Shit  bool `json:"shit"`
	}

	ts := testStruct{Hello: true, World: false, Shit: true}
	s, err := marshalLabels(ts)
	require.NoError(t, err)
	assert.Equal(t, "hello,shit", s)
}

func TestUnmarshalLabels(t *testing.T) {
	type testStruct struct {
		Hello bool `json:"hello"`
		World bool `json:"world"`
		Shit  bool `json:"shit"`
	}

	var ts testStruct

	s := "hello,world"
	err := unmarshalLabels(s, &ts)
	require.NoError(t, err)
	assert.True(t, ts.Hello)
	assert.True(t, ts.World)
	assert.False(t, ts.Shit)
}

func TestUniversalWorkload_Set(t *testing.T) {
	w := &UniversalWorkload{}
	err := w.Set("test-cluster/test-ns/deployment/whoa?no_check")
	require.NoError(t, err)
	assert.Equal(t, "test-cluster", w.Cluster)
	assert.Equal(t, "test-ns", w.Namespace)
	assert.Equal(t, "deployment", w.Type)
	assert.Equal(t, "whoa", w.Name)
	assert.Equal(t, "whoa", w.Container)
	assert.True(t, w.Labels.NoCheck)
	assert.False(t, w.Labels.Init)

	w = &UniversalWorkload{}
	err = w.Set("test-cluster/test-ns/deployment/whoa/whoa2?no_check,init")
	require.NoError(t, err)
	assert.Equal(t, "test-cluster", w.Cluster)
	assert.Equal(t, "test-ns", w.Namespace)
	assert.Equal(t, "deployment", w.Type)
	assert.Equal(t, "whoa", w.Name)
	assert.Equal(t, "whoa2", w.Container)
	assert.True(t, w.Labels.NoCheck)
	assert.True(t, w.Labels.Init)
}
