package nl

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"sync"
)

const tokenHeaderName = "Badsec-Authentication-Token"
const checksumHeaderName = "X-Request-Checksum"

type NOCList struct {
	baseURL string
	client  httpClient
	tokenMu sync.Mutex
	token   string
}

// httpClient has a single method that the standard http.Client can fulfill. This is abstraction is
// used to mock the HTTP calls in tests.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func New() *NOCList {
	return &NOCList{
		baseURL: "http://localhost:8888",
		client:  &http.Client{},
	}
}

// Fetch will handle all the logic to robustly get the NOC VIP list, including authentication and
// retries.
func (n *NOCList) Fetch() ([]string, error) {
	err := n.getAuthToken()
	if err != nil {
		return []string{}, err
	}
	return n.getUsersList()
}

func (n *NOCList) getAuthToken() error {
	n.tokenMu.Lock()
	defer n.tokenMu.Unlock()
	if n.token != "" {
		return nil
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", n.baseURL, "/auth"), nil)
	if err != nil {
		return err
	}
	resp, err := n.client.Do(req)
	if err != nil {
		// TODO: retry
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP Status %d: %s", resp.StatusCode, resp.Body)
	}
	n.token = resp.Header.Get(tokenHeaderName)
	return nil
}

func (n *NOCList) getUsersList() ([]string, error) {
	reqPath := "/users"
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", n.baseURL, reqPath), nil)
	if err != nil {
		return []string{}, err
	}
	req.Header.Add(checksumHeaderName, fmt.Sprintf("%s", n.reqChecksum(reqPath)))
	vipResp, err := n.client.Do(req)
	if err != nil {
		// TODO: retry
		return []string{}, err
	}
	defer vipResp.Body.Close()
	if vipResp.StatusCode != 200 {
		body, _ := io.ReadAll(vipResp.Body)
		return []string{}, fmt.Errorf("HTTP Status %d: %s", vipResp.StatusCode, body)
	}
	return n.parseVIPs(vipResp.Body), nil
}

func (n *NOCList) reqChecksum(reqPath string) string {
	cs := n.token + reqPath
	var sha = sha256.Sum256([]byte(cs))
	return hex.EncodeToString(sha[:])
}

func (n *NOCList) parseVIPs(vipRead io.ReadCloser) []string {
	scanner := bufio.NewScanner(vipRead)
	var vips []string
	for scanner.Scan() {
		vips = append(vips, scanner.Text())
	}
	return vips
}
