package client

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// OCSAuthConfig holds OCS web URL for authentication
type OCSAuthConfig struct {
	OCSURL string
}

func LoadOCSAuthConfig() OCSAuthConfig {
	// You can set OCS_URL in env, fallback to default
	ocsURL := os.Getenv("OCS_URL")
	if ocsURL == "" {
		ocsURL = "http://192.168.88.20/ocsreports/"
	}
	return OCSAuthConfig{OCSURL: ocsURL}
}

// AuthenticateOCSWeb tries to login to OCS web and returns username if valid, else error
func AuthenticateOCSWeb(ocsURL, username, password string) error {
	// 1. Get new PHPSESSID
	skipTLS := false
	if strings.HasPrefix(ocsURL, "https://") {
		skipTLS = true
	}
	tr := &http.Transport{}
	if skipTLS {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	client := &http.Client{Timeout: 10 * time.Second, Transport: tr}
	resp, err := client.Get(ocsURL)
	if err != nil {
		return fmt.Errorf("gagal akses OCS web: %w", err)
	}
	defer resp.Body.Close()
	var phpsessid string
	for _, c := range resp.Cookies() {
		if c.Name == "PHPSESSID" {
			phpsessid = c.Value
			break
		}
	}
	if phpsessid == "" {
		return errors.New("tidak dapat PHPSESSID dari OCS web")
	}

	// 2. POST login
	loginURL := ocsURL
	if !strings.HasSuffix(loginURL, "/index.php") {
		loginURL = strings.TrimRight(ocsURL, "/") + "/index.php"
	}
	data := url.Values{}
	data.Set("LOGIN", username)
	data.Set("PASSWD", password)
	data.Set("Valid_CNX", "Send")
	req, err := http.NewRequest("POST", loginURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("gagal buat request login: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("PHPSESSID=%s", phpsessid))

	resp2, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("gagal login ke OCS web: %w", err)
	}
	defer resp2.Body.Close()
	body, _ := ioutil.ReadAll(resp2.Body)
	if !bytes.Contains(body, []byte("My dashboard")) {
		return errors.New("Login to OCS Failed (Wrong username/password credentials)")
	}
	return nil
}
