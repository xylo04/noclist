package nl

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
)

// TODO: command line arg?
const baseURL string = "http://localhost:8888"

// Fetch will handle all the logic to robustly get the NOC VIP list, including authentication and
// retries.
func Fetch() ([]string, error) {
	token, err := getAuthToken()
	if err != nil {
		return []string{}, err
	}
	return getUsersList(token)
}

func getAuthToken() (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/auth", baseURL))
	if err != nil {
		// TODO: retry
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP Status %d: %s", resp.StatusCode, resp.Body)
	}
	token := resp.Header.Get("Badsec-Authentication-Token")
	return token, nil
}

func getUsersList(token string) ([]string, error) {
	reqPath := "/users"
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", baseURL, reqPath), nil)
	if err != nil {
		return []string{}, err
	}
	req.Header.Add("X-Request-Checksum", fmt.Sprintf("%s", reqChecksum(token, reqPath)))
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
	return parseVIPs(vipResp.Body), nil
}

func reqChecksum(token string, reqPath string) string {
	var sha = sha256.Sum256([]byte(token + reqPath))
	return hex.EncodeToString(sha[:])
}

func parseVIPs(vipRead io.ReadCloser) []string {
	scanner := bufio.NewScanner(vipRead)
	var vips []string
	for scanner.Scan() {
		vips = append(vips, scanner.Text())
	}
	return vips
}
