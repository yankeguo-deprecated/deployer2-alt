package main

type UniversalResourceList struct {
	CPU *UniversalResource `yaml:"cpu"`
	MEM *UniversalResource `yaml:"mem"`
}
