package delivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSpikeSignalWireSendIsMockableAtHTTP: the provider call is fully observable against a
// local stub — method, path, basic auth, the four form fields — and the sid comes back.
func TestSpikeSignalWireSendIsMockableAtHTTP(t *testing.T) {
	var got struct {
		method, path, user, pass string
		form                     map[string]string
	}
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method, got.path = r.Method, r.URL.Path
		got.user, got.pass, _ = r.BasicAuth()
		_ = r.ParseForm()
		got.form = map[string]string{}
		for _, k := range []string{"From", "To", "Body", "MediaUrl"} {
			got.form[k] = r.PostForm.Get(k)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid":"SM-spike-e2-0001","status":"queued","error_code":null,"error_message":null}`))
	}))
	defer stub.Close()

	s := SignalWire{BaseURL: stub.URL, ProjectID: "proj-spike", APIToken: "tok-spike", From: "+18005550100", HTTP: stub.Client()}
	sid, err := s.SendMMS(context.Background(), "+17735559930", "https://hq.yumyums.kitchen/r/spike.png", "Your Yumyums code")
	if err != nil {
		t.Fatalf("SendMMS: %v", err)
	}
	if sid != "SM-spike-e2-0001" {
		t.Errorf("sid = %q", sid)
	}
	if got.method != http.MethodPost || got.path != "/api/laml/2010-04-01/Accounts/proj-spike/Messages.json" {
		t.Errorf("request = %s %s", got.method, got.path)
	}
	if got.user != "proj-spike" || got.pass != "tok-spike" {
		t.Errorf("basic auth = %q:%q", got.user, got.pass)
	}
	for k, want := range map[string]string{"From": "+18005550100", "To": "+17735559930", "MediaUrl": "https://hq.yumyums.kitchen/r/spike.png", "Body": "Your Yumyums code"} {
		if got.form[k] != want {
			t.Errorf("form %s = %q, want %q", k, got.form[k], want)
		}
	}
	t.Logf("SPIKE-E2-1c: %s %s auth=%s form=%v sid=%s", got.method, got.path, got.user, got.form, sid)
}

// TestSpikeSignalWireRefusalIsAnError: a 401 from the provider is an error carrying the body,
// never a silent "sent".
func TestSpikeSignalWireRefusalIsAnError(t *testing.T) {
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":20003,"message":"Authenticate"}`))
	}))
	defer stub.Close()
	s := SignalWire{BaseURL: stub.URL, ProjectID: "p", APIToken: "bad", From: "+18005550100", HTTP: stub.Client()}
	sid, err := s.SendMMS(context.Background(), "+17735559930", "https://x/y.png", "")
	if err == nil || sid != "" {
		t.Fatalf("want an error and no sid, got sid=%q err=%v", sid, err)
	}
	if !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "Authenticate") {
		t.Errorf("error does not carry status and body: %v", err)
	}
	t.Logf("SPIKE-E2-1c-neg: %v", err)
}
