package nl

import (
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
		t.Log("Fetch failed: %w", err)
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

const testToken = "12345"
const testChecksum = "c20acb14a3d3339b9e92daebb173e41379f9f2fad4aa6a6326a696bd90c67419"

type mockClient struct {
	authCalledCount  uint
	usersCalledCount uint
}

func (m *mockClient) Do(req *http.Request) (*http.Response, error) {
	if req.URL.Path == "/auth" {
		m.authCalledCount++
		resp := m.makeRespBody(200, "not used")
		resp.Header.Add(tokenHeaderName, testToken)
		return resp, nil
	}

	if req.URL.Path == "/users" {
		m.usersCalledCount++
		actualChecksum := req.Header.Get(checksumHeaderName)
		if actualChecksum != testChecksum {
			return m.makeRespBody(403, fmt.Sprintf("Bad checksum (got %s)", actualChecksum)), nil
		}
		return m.makeRespBody(200, "4\n5\n6"), nil
	}

	return &http.Response{StatusCode: 404}, nil
}

func (m *mockClient) makeRespBody(code int, text string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(text)),
		Header:     map[string][]string{},
	}
}
