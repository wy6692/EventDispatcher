package wechat

import "testing"

func TestParseMessageAndDedupeKey(t *testing.T) {
	body := []byte(`<xml>
<ToUserName><![CDATA[gh_test]]></ToUserName>
<FromUserName><![CDATA[openid]]></FromUserName>
<CreateTime>1710000000</CreateTime>
<MsgType><![CDATA[event]]></MsgType>
<Event><![CDATA[CLICK]]></Event>
<EventKey><![CDATA[MENU_ORDER]]></EventKey>
</xml>`)
	fields, err := ParseFields(body)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := MessageFromFields(fields)
	if err != nil {
		t.Fatal(err)
	}
	if msg.MsgType != "event" || msg.Event != "CLICK" || msg.EventKey != "MENU_ORDER" || msg.CreateTime != 1710000000 {
		t.Fatalf("%+v", msg)
	}
	if msg.MsgID != "" {
		t.Fatal("event should not have MsgId")
	}
	key := DedupeKey("wx_app", msg)
	if key != "wx_app\x1fopenid\x1f1710000000\x1fevent\x1fCLICK\x1fMENU_ORDER" {
		t.Fatalf("dedupe = %q", key)
	}

	msg.MsgID = "99"
	if DedupeKey("wx_app", msg) != "wx_app\x1f99" {
		t.Fatalf("msgid dedupe = %q", DedupeKey("wx_app", msg))
	}
}
