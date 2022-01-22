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

type NOCList struct {
	baseURL string
	tokenMu sync.Mutex
	token   string
}

func New() *NOCList {
	return &NOCList{
		baseURL: "http://localhost:8888",
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
	if n.token != "" {
		return nil
	}
	resp, err := http.Get(fmt.Sprintf("%s/auth", n.baseURL))
	if err != nil {
		// TODO: retry
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP Status %d: %s", resp.StatusCode, resp.Body)
	}
	n.token = resp.Header.Get("Badsec-Authentication-Token")
	n.tokenMu.Unlock()
	return nil
}

func (n *NOCList) getUsersList() ([]string, error) {
	reqPath := "/users"
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", n.baseURL, reqPath), nil)
	if err != nil {
		return []string{}, err
	}
	req.Header.Add("X-Request-Checksum", fmt.Sprintf("%s", n.reqChecksum(n.token, reqPath)))
	client := &http.Client{}
	vipResp, err := client.Do(req)
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

func (n *NOCList) reqChecksum(token string, reqPath string) string {
	var sha = sha256.Sum256([]byte(token + reqPath))
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
