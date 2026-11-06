package wechat

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestEncryptRoundTrip(t *testing.T) {
	key := testAESKey()
	msg := []byte(`<xml><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[subscribe]]></Event></xml>`)
	enc, err := Encrypt(key, "wx_app", msg)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Decrypt(key, "wx_app", enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plain, msg) {
		t.Fatalf("plain = %q", plain)
	}
	if _, err := Decrypt(key, "wx_other", enc); !errors.Is(err, ErrAppIDMismatch) {
		t.Fatalf("err = %v", err)
	}
	if _, err := Decrypt(key, "wx_app", "%%%"); !errors.Is(err, ErrInvalidCipher) {
		t.Fatalf("err = %v", err)
	}
}

func TestCleanBase64RestoresPlus(t *testing.T) {
	if got := CleanBase64("abc def"); got != "abc+def" {
		t.Fatalf("got %q", got)
	}
}

func TestSignatureSortsParts(t *testing.T) {
	got := Signature("token", "2", "b", "a")
	want := Signature("a", "b", "2", "token")
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
	if ValidSignature(got, strings.ToUpper(got)) != true {
		t.Fatal("expected case-insensitive match")
	}
	if ValidSignature(got, got+"aa") {
		t.Fatal("different length must fail")
	}
}

func testAESKey() string {
	raw := bytes.Repeat([]byte{7}, 32)
	return strings.TrimSuffix(base64.StdEncoding.EncodeToString(raw), "=")
}
