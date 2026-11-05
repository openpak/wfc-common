package gperror

import (
	"strings"
	"testing"
	"unicode/utf16"

	"openpak/wfc-common/common"
)

func decodeErrMsg(t *testing.T, msg string) string {
	t.Helper()
	i := strings.Index(msg, `\wl:errmsg\`)
	if i < 0 {
		return ""
	}
	v := msg[i+len(`\wl:errmsg\`):]
	v = v[:strings.Index(v, `\`)]
	b, err := common.Base64DwcEncoding.DecodeString(v)
	if err != nil {
		t.Fatal(err)
	}
	u := make([]uint16, len(b)/2)
	for k := range u {
		u[k] = uint16(b[2*k])<<8 | uint16(b[2*k+1])
	}
	return string(utf16.Decode(u))
}

func TestNotLinkedMarioKartWiiShowsCodeAndNumber(t *testing.T) {
	for _, lang := range []byte{LangEnglish, LangEnglishEU, LangFrench, LangFrenchEU, LangGerman, LangJapanese, LangKorean} {
		msg, code := NotLinkedError("mariokartwii", "ABCD-EFGH", 482913, 0).GetMessageTranslate("mariokartwii", 0, lang, 0, 0)
		if code != 22100 || !strings.Contains(msg, `\wl:err\22100`) {
			t.Fatalf("lang %x: code %d msg %q", lang, code, msg)
		}
		text := decodeErrMsg(t, msg)
		if !strings.Contains(text, "ABCD-EFGH") || !strings.Contains(text, "482913") || !strings.Contains(text, "22100") || !strings.Contains(text, "openpak.org/link") {
			t.Fatalf("lang %x: text %q", lang, text)
		}
		if n := len(utf16.Encode([]rune(text))); n >= 256 {
			t.Fatalf("lang %x: %d units, payload buffer is 256", lang, n)
		}
	}
}

func TestNotLinkedOtherGamesShowTheNumber(t *testing.T) {
	msg, code := NotLinkedError("tetrisds", "ABCD-EFGH", 482913, 0).GetMessageTranslate("smashbrawlx", 0, LangEnglish, 0, 0)
	if code != 482913 || !strings.Contains(msg, `\wl:err\482913`) || strings.Contains(msg, "wl:errmsg") || strings.Contains(msg, "ABCD") {
		t.Fatalf("code %d msg %q", code, msg)
	}
	if !strings.Contains(msg, `\fatal\`) {
		t.Fatal("must be fatal")
	}
}

func TestNotLinkedDSErr(t *testing.T) {
	msg, _ := NotLinkedError("mariokartds", "ABCD-EFGH", 482913, 263).GetMessageTranslate("mariokartds", 0, LangEnglish, 0, 0)
	if !strings.Contains(msg, `\err\263\`) {
		t.Fatalf("msg %q", msg)
	}
}
