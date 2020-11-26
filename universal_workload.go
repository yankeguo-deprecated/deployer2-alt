package main

import (
	"errors"
	"strings"
)

var (
	knownWorkloadTypes = []string{
		"deployment",
		"statefulset",
		"daemonset",
		"cronjob",
		"deploy",
		"ds",
		"sts",
	}
)

func sanitizeWorkloadName(s string) string {
	return strings.TrimSpace(
		strings.ToLower(
			strings.ReplaceAll(
				strings.ReplaceAll(s, ".", "-"),
				"_", "-")))
}

// UniversalWorkload 在多种工作负载类型下，引用其中固定的容器，默认容器名与工作负载名相等（Rancher 惯例）
type UniversalWorkload struct {
	Cluster   string
	Namespace string
	Type      string
	Name      string
	Container string
	IsInit    bool
}

func (w UniversalWorkload) String() string {
	sb := &strings.Builder{}
	sb.WriteString(w.Cluster)
	sb.WriteRune('/')
	sb.WriteString(w.Namespace)
	sb.WriteRune('/')
	sb.WriteString(w.Name)
	sb.WriteRune('/')
	sb.WriteString(w.Container)
	if w.IsInit {
		sb.WriteRune('!')
	}
	return sb.String()
}

func (w *UniversalWorkload) Set(s string) error {
	splits := strings.Split(s, "/")
	if len(splits) != 4 && len(splits) != 5 {
		return errors.New("目标工作负载参数格式不正确")
	}
	w.Cluster,
		w.Namespace,
		w.Type,
		w.Name = sanitizeWorkloadName(splits[0]),
		sanitizeWorkloadName(splits[1]),
		sanitizeWorkloadName(splits[2]),
		sanitizeWorkloadName(splits[3])
	if len(splits) == 5 {
		w.Container = sanitizeWorkloadName(splits[4])
	} else {
		w.Container = w.Name
	}
	w.IsInit = strings.HasSuffix(w.Container, "!")
	w.Container = strings.TrimSuffix(w.Container, "!")
	for _, kt := range knownWorkloadTypes {
		if kt == w.Type {
			return nil
		}
	}
	return errors.New("目标工作负载参数指定了未知的类型")
}

type UniversalWorkloads []UniversalWorkload

func (ws UniversalWorkloads) String() string {
	sb := &strings.Builder{}
	for _, w := range ws {
		if sb.Len() > 0 {
			sb.WriteRune(',')
		}
		sb.WriteString(w.String())
	}
	return sb.String()
}

func (ws *UniversalWorkloads) Set(s string) error {
	w := &UniversalWorkload{}
	if err := w.Set(s); err != nil {
		return err
	} else {
		*ws = append(*ws, *w)
		return nil
	}
}
