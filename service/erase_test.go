package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const acct = "0b0b0b0b-1111-4222-8333-444444444444"

func call(h http.Handler, method, path, auth string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, nil)
	if auth != "" {
		r.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestEraseHandler(t *testing.T) {
	n := 0
	h := eraseHandler("wfc", "secret", func(ctx context.Context, id string) (int, error) { n++; return 7, nil })
	p := "/internal/accounts/" + acct + "/data"
	for _, a := range []string{"", "Bearer wrong", "secret"} {
		if w := call(h, "DELETE", p, a); w.Code != 401 {
			t.Fatalf("auth %q: %d", a, w.Code)
		}
	}
	if w := call(h, "DELETE", "/internal/accounts/not-a-uuid/data", "Bearer secret"); w.Code != 404 {
		t.Fatalf("bad uuid: %d", w.Code)
	}
	if n != 0 {
		t.Fatal("erase ran without valid auth")
	}
	w := call(h, "DELETE", p, "Bearer secret")
	if w.Code != 200 || w.Body.String() != "{\"rows\":7,\"service\":\"wfc\"}\n" || n != 1 {
		t.Fatalf("%d %q %d", w.Code, w.Body.String(), n)
	}
	if w := call(h, "GET", p, "Bearer secret"); w.Code != 405 {
		t.Fatalf("get: %d", w.Code)
	}
}

func TestEraseHandlerUnconfigured(t *testing.T) {
	h := eraseHandler("wfc", "", func(context.Context, string) (int, error) { t.Fatal("ran"); return 0, nil })
	if w := call(h, "DELETE", "/internal/accounts/"+acct+"/data", "Bearer "); w.Code != 503 {
		t.Fatalf("%d", w.Code)
	}
}
