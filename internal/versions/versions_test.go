package versions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openkcm/checker/internal/config"
)

func TestQueryNoResources(t *testing.T) {
	got := Query(context.Background(), &config.Versions{Resources: nil})

	if len(got) != 0 {
		t.Errorf("expected empty response, got %v", got)
	}
}

func TestQuerySuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"version":"1.2.3"}`))
	}))
	defer srv.Close()

	cfg := &config.Versions{
		Timeout:   2 * time.Second,
		Resources: []*config.ServiceResource{{Name: "svc", URL: srv.URL}},
	}

	got := Query(context.Background(), cfg)

	res, ok := got["svc"].(*Response)
	if !ok {
		t.Fatalf("expected *Response for svc, got %T", got["svc"])
	}

	if res.Status != OK {
		t.Errorf("status = %q, want %q", res.Status, OK)
	}

	result, ok := res.Result.(map[string]any)
	if !ok || result["version"] != "1.2.3" {
		t.Errorf("result = %v, want version 1.2.3", res.Result)
	}
}

func TestQueryCallError(t *testing.T) {
	cfg := &config.Versions{
		Timeout:   time.Second,
		Resources: []*config.ServiceResource{{Name: "svc", URL: "http://127.0.0.1:0/unreachable"}},
	}

	got := Query(context.Background(), cfg)

	res, ok := got["svc"].(*Response)
	if !ok {
		t.Fatalf("svc has type %T, want *Response", got["svc"])
	}

	if res.Status != NOTOK {
		t.Errorf("status = %q, want %q", res.Status, NOTOK)
	}

	if res.Error == nil {
		t.Fatal("expected error response, got nil")
	}
}

func TestUnmarshalValueSuccess(t *testing.T) {
	res := &Response{Status: OK}

	unmarshalValue(`{"a":"b"}`, res)

	if res.Status != OK {
		t.Errorf("status = %q, want OK", res.Status)
	}

	result, ok := res.Result.(map[string]any)
	if !ok {
		t.Fatalf("result has type %T, want map[string]any", res.Result)
	}

	if result["a"] != "b" {
		t.Errorf("result = %v, want a=b", res.Result)
	}
}

func TestUnmarshalValueExtractError(t *testing.T) {
	res := &Response{Status: OK}

	// A base64(...) wrapper with an invalid payload fails extraction/decoding.
	unmarshalValue("base64(@@not-base64@@)", res)

	if res.Status != NOTOK {
		t.Errorf("status = %q, want NOT OK", res.Status)
	}

	if res.Result != nil {
		t.Errorf("result = %v, want nil", res.Result)
	}

	if res.Error == nil {
		t.Fatal("expected error response, got nil")
	}
}

func TestUnmarshalValueInvalidJSON(t *testing.T) {
	res := &Response{Status: OK}

	// A valid non-empty string that is not valid JSON object.
	unmarshalValue("not-json", res)

	if res.Status != NOTOK {
		t.Errorf("status = %q, want NOT OK", res.Status)
	}

	if res.Result != nil {
		t.Errorf("result = %v, want nil", res.Result)
	}

	if res.Error == nil {
		t.Fatal("expected error response, got nil")
	}
}

func TestCallSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer srv.Close()

	body, err := call(context.Background(), srv.Client(), &config.ServiceResource{URL: srv.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(body) != "payload" {
		t.Errorf("body = %q, want payload", body)
	}
}

func TestCallBadURL(t *testing.T) {
	_, err := call(context.Background(), http.DefaultClient, &config.ServiceResource{URL: "://bad-url"})
	if err == nil {
		t.Fatal("expected error for malformed URL")
	}
}

func TestCallDoError(t *testing.T) {
	_, err := call(context.Background(), &http.Client{Timeout: time.Millisecond}, &config.ServiceResource{URL: "http://127.0.0.1:0/x"})
	if err == nil {
		t.Fatal("expected error from client.Do")
	}
}
