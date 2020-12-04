package main

import (
	"encoding/json"
	"errors"
	"sort"
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

func marshalLabels(v interface{}) (s string, err error) {
	var buf []byte
	if buf, err = json.Marshal(v); err != nil {
		return
	}
	var m map[string]bool
	if err = json.Unmarshal(buf, &m); err != nil {
		return
	}
	var ks []string
	for k, v := range m {
		if !v {
			continue
		}
		ks = append(ks, k)
	}
	sort.Strings(ks)
	s = strings.Join(ks, ",")
	return
}

func unmarshalLabels(s string, v interface{}) (err error) {
	splits := strings.Split(s, ",")
	m := map[string]bool{}
	for _, split := range splits {
		m[strings.TrimSpace(split)] = true
	}
	var buf []byte
	if buf, err = json.Marshal(m); err != nil {
		return
	}
	err = json.Unmarshal(buf, v)
	return
}

// UniversalWorkload 在多种工作负载类型下，引用其中固定的容器，默认容器名与工作负载名相等（Rancher 惯例）
type UniversalWorkload struct {
	Cluster   string
	Namespace string
	Type      string
	Name      string
	Container string
	Labels    struct {
		Init    bool `json:"init,omitempty"`
		NoCheck bool `json:"no_check,omitempty"`
	}
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
	l, _ := marshalLabels(w.Labels)
	if l != "" {
		sb.WriteRune('?')
		sb.WriteString(l)
	}
	return sb.String()
}

func (w *UniversalWorkload) Set(s string) error {
	labelSplits := strings.Split(s, "?")
	if len(labelSplits) == 2 {
		s = labelSplits[0]
		if err := unmarshalLabels(labelSplits[1], &w.Labels); err != nil {
			return err
		}
	}
	itemSplits := strings.Split(s, "/")
	if len(itemSplits) != 4 && len(itemSplits) != 5 {
		return errors.New("目标工作负载参数格式不正确")
	}
	w.Cluster,
		w.Namespace,
		w.Type,
		w.Name = sanitizeWorkloadName(itemSplits[0]),
		sanitizeWorkloadName(itemSplits[1]),
		sanitizeWorkloadName(itemSplits[2]),
		sanitizeWorkloadName(itemSplits[3])
	if len(itemSplits) == 5 {
		w.Container = sanitizeWorkloadName(itemSplits[4])
	} else {
		w.Container = w.Name
	}
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
