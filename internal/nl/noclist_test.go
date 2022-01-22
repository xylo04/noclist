package nl

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHappyPath(t *testing.T) {
	fetcher := New()
	m := &mockClient{}
	fetcher.client = m

	vips, err := fetcher.Fetch()
	if err != nil {
		t.Logf("Fetch failed: %v", err)
		t.FailNow()
	}
	if len(vips) < 1 {
		t.Log("NOC list was empty")
		t.FailNow()
	}
	if m.authCalledCount != 1 || m.usersCalledCount != 1 {
		t.Logf("unexpected number of HTTP calls, expected (1,1), actual (%d,%d)", m.authCalledCount, m.usersCalledCount)
		t.FailNow()
	}
}

func TestRetry(t *testing.T) {
	fetcher := New()
	m := &mockClient{
		customAuthResponses: []*http.Response{
			makeRespBody(500, "Internal Server Error"),
			makeRespBody(500, "Internal Server Error"),
			// then back to happy
		},
		customUsersResponses: []*http.Response{
			makeRespBody(503, "Overloaded, try again later"),
			makeRespBody(500, "Internal Server Error"),
			// then back to happy
		},
	}
	fetcher.client = m

	vips, err := fetcher.Fetch()
	if err != nil {
		t.Logf("Fetch failed: %v", err)
		t.FailNow()
	}
	if len(vips) < 1 {
		t.Log("NOC list was empty")
		t.FailNow()
	}
	if m.authCalledCount != 3 || m.usersCalledCount != 3 {
		t.Logf("unexpected number of HTTP calls, expected (3,3), actual (%d,%d)", m.authCalledCount, m.usersCalledCount)
		t.FailNow()
	}
}

func TestRetryFail(t *testing.T) {
	fetcher := New()
	m := &mockClient{
		customAuthResponses: []*http.Response{
			makeRespBody(500, "Internal Server Error"),
			makeRespBody(500, "Internal Server Error"),
			// then back to happy
		},
		customUsersResponses: []*http.Response{
			makeRespBody(503, "Overloaded, try again later"),
			makeRespBody(500, "Internal Server Error"),
			makeRespBody(500, "Internal Server Error"), // should fail after this
			makeRespBody(500, "Internal Server Error"),
		},
	}
	fetcher.client = m

	_, err := fetcher.Fetch()
	if !errors.Is(err, TooManyRetries) {
		t.Log("Expected a TooManyRetries error, but that wasn't thrown")
		t.FailNow()
	}
	if m.authCalledCount != 3 || m.usersCalledCount != 3 {
		t.Logf("unexpected number of HTTP calls, expected (3,3), actual (%d,%d)", m.authCalledCount, m.usersCalledCount)
		t.FailNow()
	}
}

const testToken = "12345"
const testChecksum = "c20acb14a3d3339b9e92daebb173e41379f9f2fad4aa6a6326a696bd90c67419"

type mockClient struct {
	authCalledCount      uint
	usersCalledCount     uint
	customAuthResponses  []*http.Response
	customUsersResponses []*http.Response
}

func (m *mockClient) Do(req *http.Request) (*http.Response, error) {
	if req.URL.Path == "/auth" {
		m.authCalledCount++
		if len(m.customAuthResponses) > 0 {
			customResp := m.customAuthResponses[0]
			m.customAuthResponses = m.customAuthResponses[1:]
			return customResp, nil
		}
		resp := makeRespBody(200, "not used")
		resp.Header.Add(tokenHeaderName, testToken)
		return resp, nil
	}

	if req.URL.Path == "/users" {
		m.usersCalledCount++
		if len(m.customUsersResponses) > 0 {
			customResp := m.customUsersResponses[0]
			m.customUsersResponses = m.customUsersResponses[1:]
			return customResp, nil
		}
		actualChecksum := req.Header.Get(checksumHeaderName)
		if actualChecksum != testChecksum {
			return makeRespBody(403, fmt.Sprintf("Bad checksum (got %s)", actualChecksum)), nil
		}
		return makeRespBody(200, "4\n5\n6"), nil
	}

	return &http.Response{StatusCode: 404}, nil
}

func makeRespBody(code int, text string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(text)),
		Header:     map[string][]string{},
	}
}
