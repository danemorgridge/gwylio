package elastic

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFailoverHTTPRequestHitsFirstGoodRequest(t *testing.T) {

	firstTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "firstTestServer")
	}))
	defer firstTestServer.Close()

	secondTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "secondTestServer")
	}))
	defer secondTestServer.Close()

	hosts := []string{firstTestServer.URL, secondTestServer.URL}

	body, err := failoverHTTPRequest(hosts, "GET", "/", nil)
	if err != nil {
		t.Fail()
		t.Log(err)
	}

	if !strings.Contains(string(body), "firstTestServer") {
		t.Fail()
		t.Log("Request should have been handled by firstTestServer, was ", string(body))
	}
}

func TestFailoverHTTPRequestSkipsBadRequest(t *testing.T) {

	firstTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer firstTestServer.Close()

	secondTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "secondTestServer")
	}))
	defer secondTestServer.Close()

	hosts := []string{firstTestServer.URL, secondTestServer.URL}

	body, err := failoverHTTPRequest(hosts, "GET", "/", nil)
	if err != nil {
		t.Fail()
		t.Log(err)
	}

	if !strings.Contains(string(body), "secondTestServer") {
		t.Fail()
		t.Log("Request should have been handled by secondTestServer, was ", string(body))
	}
}

func TestFailoverHTTPRequestFailsOnNoGoodServers(t *testing.T) {

	firstTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer firstTestServer.Close()

	secondTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer secondTestServer.Close()

	hosts := []string{firstTestServer.URL, secondTestServer.URL}

	_, err := failoverHTTPRequest(hosts, "GET", "/", nil)
	if err == nil {
		t.Fail()
		t.Log("Request should have errored out")
	}
}
