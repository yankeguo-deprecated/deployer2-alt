package main

import (
	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	defaultUniversalCheck = UniversalCheck{
		Port:     8080,
		Delay:    60,
		Interval: 15,
		Success:  1,
		Failure:  2,
		Timeout:  5,
	}
)

type UniversalCheck struct {
	Port     int    `yaml:"port"`
	Path     string `yaml:"path"`
	Delay    int    `yaml:"delay"`
	Interval int    `yaml:"interval"`
	Success  int    `yaml:"success"`
	Failure  int    `yaml:"failure"`
	Timeout  int    `yaml:"timeout"`
}

func (c UniversalCheck) GenerateReadinessProbe() *corev1.Probe {
	_ = mergo.Merge(&c, defaultUniversalCheck)
	if c.Path == "" {
		return nil
	}
	b := &corev1.Probe{
		InitialDelaySeconds: int32(c.Delay),
		TimeoutSeconds:      int32(c.Timeout),
		PeriodSeconds:       int32(c.Interval),
		SuccessThreshold:    int32(c.Success),
		FailureThreshold:    int32(c.Failure),
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   c.Path,
				Port:   intstr.FromInt(c.Port),
				Scheme: "HTTP",
			},
		},
	}
	return b
}

func (c UniversalCheck) GenerateLivenessProbe() *corev1.Probe {
	p := c.GenerateReadinessProbe()
	if p != nil {
		// LivenessProbe 强制要求 SuccessThreshold = 1
		p.SuccessThreshold = 1
	}
	return p
}
