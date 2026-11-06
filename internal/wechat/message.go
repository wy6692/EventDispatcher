package wechat

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Message 是解密并解析后的一条微信推送。
type Message struct {
	ToUser     string
	FromUser   string
	CreateTime int64
	MsgType    string
	Event      string
	EventKey   string
	MsgID      string
	Fields     map[string]string
}

// ParseFields 读取扁平 XML。微信消息的字段都在 <xml> 下一层。
func ParseFields(body []byte) (map[string]string, error) {
	body = bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})
	dec := xml.NewDecoder(bytes.NewReader(body))
	fields := map[string]string{}
	var current string
	var buf strings.Builder
	depth := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("解析 XML: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if depth == 2 {
				current = t.Name.Local
				buf.Reset()
			}
		case xml.CharData:
			if depth == 2 && current != "" {
				buf.Write(t)
			}
		case xml.EndElement:
			if depth == 2 && current != "" {
				fields[current] = strings.TrimSpace(buf.String())
				current = ""
			}
			depth--
		}
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("XML 没有字段")
	}
	return fields, nil
}

// MessageFromFields 抽出路由和去重会用到的字段。
func MessageFromFields(fields map[string]string) (Message, error) {
	msgType := fields["MsgType"]
	if msgType == "" {
		return Message{}, fmt.Errorf("缺少 MsgType")
	}
	var createTime int64
	if raw := fields["CreateTime"]; raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return Message{}, fmt.Errorf("CreateTime 无效")
		}
		createTime = n
	}
	msgID := fields["MsgId"]
	if msgID == "" {
		msgID = fields["MsgID"]
	}
	copied := make(map[string]string, len(fields))
	for k, v := range fields {
		copied[k] = v
	}
	return Message{
		ToUser:     fields["ToUserName"],
		FromUser:   fields["FromUserName"],
		CreateTime: createTime,
		MsgType:    msgType,
		Event:      fields["Event"],
		EventKey:   fields["EventKey"],
		MsgID:      msgID,
		Fields:     copied,
	}, nil
}

// DedupeKey 有 MsgId 时用 appId 加 MsgId，否则用事件身份字段。
func DedupeKey(appID string, msg Message) string {
	if msg.MsgID != "" {
		return appID + "\x1f" + msg.MsgID
	}
	return strings.Join([]string{
		appID,
		msg.FromUser,
		strconv.FormatInt(msg.CreateTime, 10),
		msg.MsgType,
		msg.Event,
		msg.EventKey,
	}, "\x1f")
}
