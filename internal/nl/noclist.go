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
	resp, err := n.doWithRetry(req)
	if err != nil {
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
	vipResp, err := n.doWithRetry(req)
	if err != nil {
		return []string{}, err
	}
	defer vipResp.Body.Close()
	if vipResp.StatusCode != 200 {
		return []string{}, n.makeRespError(vipResp)
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

func (n *NOCList) doWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	for i := 0; i < 3; i++ {
		resp, err = n.client.Do(req)
		if err == nil && resp.StatusCode < 300 {
			return resp, nil
		}
		if resp.StatusCode/100 == 4 {
			// client error, don't retry
			return resp, n.makeRespError(resp)
		}
	}
	return resp, fmt.Errorf("too many retries")
}

func (n *NOCList) makeRespError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("HTTP Status %d: %s", resp.StatusCode, body)
}
