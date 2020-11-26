package main

import (
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	"strconv"
	"strings"
)

const (
	ResourceInfinity int64 = -1
)

// UniversalResource 资源配额，Request 不允许 0 值，Limit 允许 0 值代表无限制，针对 CPU 单位为 m，针对内存单位为 Mi
type UniversalResource struct {
	Request int64
	Limit   int64
}

func (l *UniversalResource) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	var s string
	if err = unmarshal(&s); err != nil {
		return
	}
	if err = l.Set(s); err != nil {
		return
	}
	return
}

func (l UniversalResource) IsZero() bool {
	return l.Request == 0 && l.Limit == 0
}

func (l UniversalResource) String() string {
	if l.Limit == ResourceInfinity {
		return fmt.Sprintf("%d:-", l.Request)
	}
	return fmt.Sprintf("%d:%d", l.Request, l.Limit)
}

func (l *UniversalResource) Set(s string) (err error) {
	splits := strings.Split(s, ":")
	if len(splits) != 2 {
		err = errors.New("资源配额格式不正确")
		return
	}
	if l.Request, err = strconv.ParseInt(splits[0], 10, 64); err != nil {
		return
	}
	if splits[1] == "-" {
		l.Limit = ResourceInfinity
	} else {
		if l.Limit, err = strconv.ParseInt(splits[1], 10, 64); err != nil {
			return
		}
	}
	if l.Request <= 0 || (l.Limit != ResourceInfinity && l.Limit < l.Request) {
		err = errors.New("资源配额格式不正确")
		return
	}
	return
}

func (l UniversalResource) AsCPU() (resource.Quantity, resource.Quantity) {
	if l.Limit == ResourceInfinity {
		return resource.MustParse(fmt.Sprintf("%dm", l.Request)),
			resource.MustParse(fmt.Sprintf("999"))
	}
	return resource.MustParse(fmt.Sprintf("%dm", l.Request)),
		resource.MustParse(fmt.Sprintf("%dm", l.Limit))
}

func (l UniversalResource) AsMEM() (resource.Quantity, resource.Quantity) {
	if l.Limit == ResourceInfinity {
		return resource.MustParse(fmt.Sprintf("%dMi", l.Request)),
			resource.MustParse(fmt.Sprintf("999Gi"))
	}
	return resource.MustParse(fmt.Sprintf("%dMi", l.Request)),
		resource.MustParse(fmt.Sprintf("%dMi", l.Limit))
}
