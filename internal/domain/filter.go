package domain

import (
	"encoding/json"
	"strings"
)

// Filter 定义订阅方事件过滤规则。
type Filter struct {
	MsgTypes         []string `json:"msgTypes,omitempty"`
	Events           []string `json:"events,omitempty"`
	EventKeyPrefixes []string `json:"eventKeyPrefixes,omitempty"`
}

// Matches 判断给定的消息类型与事件是否命中本规则。
func (f *Filter) Matches(msgType, event, eventKey string) bool {
	if f == nil {
		return true
	}
	if len(f.MsgTypes) > 0 && !containsString(f.MsgTypes, msgType) {
		return false
	}
	if len(f.Events) > 0 && !containsString(f.Events, event) {
		return false
	}
	if len(f.EventKeyPrefixes) > 0 && !hasAnyPrefix(f.EventKeyPrefixes, eventKey) {
		return false
	}
	return true
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}

func hasAnyPrefix(prefixes []string, target string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(target, prefix) {
			return true
		}
	}
	return false
}

// ParseFilter 从 JSON 字符串解析出 Filter 实体。
func ParseFilter(rawJSON string) (*Filter, error) {
	rawJSON = strings.TrimSpace(rawJSON)
	if rawJSON == "" || rawJSON == "{}" {
		return &Filter{}, nil
	}
	var f Filter
	if err := json.Unmarshal([]byte(rawJSON), &f); err != nil {
		return nil, err
	}
	return &f, nil
}
