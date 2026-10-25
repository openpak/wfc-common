package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProxyPreservesWireAndAuthenticates(t *testing.T) {
	t.Setenv("WFC_INTERNAL_KEY", strings.Repeat("k", 32))
	upstream := httptest.NewServer(Guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "sake.gs.nintendowifi.net" || r.URL.RawQuery != "x=1" {
			t.Error("wire routing changed")
		}
		w.Header().Set("X-Wire", "preserved")
		w.Write([]byte("reply"))
	})))
	defer upstream.Close()
	proxy, err := Proxy(upstream.URL, Key())
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "http://sake.gs.nintendowifi.net/path?x=1", nil)
	r.Header.Set("X-WFC-Internal-Key", "untrusted-client-value")
	w := httptest.NewRecorder()
	proxy.ServeHTTP(w, r)
	if w.Code != 200 || w.Body.String() != "reply" || w.Header().Get("X-Wire") != "preserved" {
		t.Fatal(w.Code, w.Body)
	}
	w = httptest.NewRecorder()
	Guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("unauthenticated request reached handler") })).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
	upstream.Close()
	w = httptest.NewRecorder()
	proxy.ServeHTTP(w, r)
	if w.Code != 503 {
		t.Fatal("outage did not fail closed", w.Code)
	}
}
