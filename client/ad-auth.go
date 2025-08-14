// client/ad-auth.go
package client

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

const admpCSRFCookieName = "admpcsrf"

// Config menyimpan semua konfigurasi yang dibutuhkan oleh client.
type Config struct {
	BaseURL           string
	Username          string
	EncryptedPassword string
	ReportID          string
	GenerationID      string
}

// Client adalah object yang akan menangani semua interaksi HTTP.
type Client struct {
	httpClient *http.Client
	config     Config
}

// New membuat instance baru dari Client.
func New(config Config) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat cookie jar: %v", err)
	}
	return &Client{
		httpClient: &http.Client{Jar: jar},
		config:     config,
	}, nil
}

// Login adalah method dari Client yang melakukan autentikasi.
func (c *Client) Login() error {
	// Langkah 1: GET halaman login
	resp, err := c.httpClient.Get(c.config.BaseURL + "/")
	if err != nil {
		return fmt.Errorf("gagal melakukan GET request awal: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code tidak valid saat GET awal: %d", resp.StatusCode)
	}
	// log.Println("[INFO] ADManager Plus - Initial Cookie Successfully fetched.")

	// Langkah 2: POST data login
	loginURL := c.config.BaseURL + "/j_security_check?LogoutFromSSO=true"
	loginPayload := url.Values{}
	loginPayload.Set("is_admp_pass_encrypted", "true")
	loginPayload.Set("j_username", c.config.Username)
	loginPayload.Set("j_password", c.config.EncryptedPassword)
	loginPayload.Set("domainName", "ADManager Plus Authentication")
	loginPayload.Set("AUTHRULE_NAME", "ADAuthenticator")

	req, _ := http.NewRequest("POST", loginURL, strings.NewReader(loginPayload.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gagal melakukan POST request login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login gagal dengan status code: %d", resp.StatusCode)
	}

	log.Println("[SUCCESS] AD Manager Plus - Login Success! Session is valid.")
	return nil
}

// FetchComputerReport adalah method dari Client untuk mengambil data laporan.
func (c *Client) FetchComputerReport() ([]byte, error) {
	reportURL := c.config.BaseURL + "/api/json/reports/report/getReportResultRows"
	parsedBaseURL, _ := url.Parse(c.config.BaseURL)

	var csrfToken string
	for _, cookie := range c.httpClient.Jar.Cookies(parsedBaseURL) {
		if cookie.Name == admpCSRFCookieName {
			csrfToken = cookie.Value
			break
		}
	}
	if csrfToken == "" {
		return nil, fmt.Errorf("tidak dapat menemukan cookie '%s' setelah login", admpCSRFCookieName)
	}
	// fmt.Printf("[*] Menggunakan CSRF Token dari cookie: %s\n", csrfToken)

	paramsData := map[string]interface{}{
		"pageNavigateData": map[string]interface{}{"startIndex": 1, "toIndex": 999999, "rangeList": []int{25, 50, 75, 100}, "range": 999999, "totalCount": 0, "isNavigate": false},
		"searchText":       map[string]interface{}{}, "searchCriteriaType": map[string]interface{}{}, "sortAttribId": -1,
		"sortingOrder": true, "reportResultFilter": map[string]interface{}{}, "rvcFilter": map[string]interface{}{}, "viewOf": "default",
		"dbFilterDetails": map[string]interface{}{"objectId": 3, "filters": []interface{}{}},
	}
	paramsJSON, err := json.Marshal(paramsData)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat JSON untuk params: %v", err)
	}

	reportPayload := url.Values{}
	reportPayload.Set("reportId", c.config.ReportID)
	reportPayload.Set("generationId", c.config.GenerationID)
	reportPayload.Set("params", string(paramsJSON))
	reportPayload.Set("intersect", "false")
	reportPayload.Set("admpcsrf", csrfToken)

	req, _ := http.NewRequest("POST", reportURL, strings.NewReader(reportPayload.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gagal melakukan POST request laporan: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gagal mengambil laporan, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// log.Println("[SUCCESS] AD Manager Plus - Berhasil mengambil data laporan!")
	return io.ReadAll(resp.Body)
}
