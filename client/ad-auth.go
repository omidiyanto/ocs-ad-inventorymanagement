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
	"sync"
	"time"
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
	mu         sync.RWMutex
}

// IsSessionValid checks if the ADManager Plus session is still valid by checking the existence of the CSRF cookie.
func (c *Client) IsSessionValid() bool {
	parsedBaseURL, err := url.Parse(c.config.BaseURL)
	if err != nil {
		return false
	}
	for _, cookie := range c.httpClient.Jar.Cookies(parsedBaseURL) {
		if cookie.Name == admpCSRFCookieName && cookie.Value != "" {
			return true
		}
	}
	return false
}

// GetLatestGenerationID posts to generateReport and returns the latest generationId for the given reportId and params.
func (c *Client) GetLatestGenerationID(reportId string, params string) (string, error) {
	genURL := c.config.BaseURL + "/api/json/reports/report/generateReport"
	parsedBaseURL, _ := url.Parse(c.config.BaseURL)
	var csrfToken string
	for _, cookie := range c.httpClient.Jar.Cookies(parsedBaseURL) {
		if cookie.Name == admpCSRFCookieName {
			csrfToken = cookie.Value
			break
		}
	}
	if csrfToken == "" {
		return "", fmt.Errorf("tidak dapat menemukan cookie '%s' setelah login", admpCSRFCookieName)
	}
	payload := url.Values{}
	payload.Set("reportId", reportId)
	payload.Set("params", params)
	payload.Set("admpcsrf", csrfToken)
	req, _ := http.NewRequest("POST", genURL, strings.NewReader(payload.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal POST generateReport: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gagal generateReport, status: %d, body: %s", resp.StatusCode, string(body))
	}
	body, _ := io.ReadAll(resp.Body)
	// Logging isi response untuk debug
	fmt.Printf("[DEBUG] Response generateReport: %s\n", string(body))
	// Validasi jika response bukan JSON
	if strings.Contains(string(body), "<") {
		return "", fmt.Errorf("response generateReport bukan JSON, kemungkinan error dari server: %s", string(body))
	}
	var genResp map[string]interface{}
	if err := json.Unmarshal(body, &genResp); err != nil {
		return "", fmt.Errorf("gagal parsing response generateReport: %v. Body: %s", err, string(body))
	}
	if genId, ok := genResp["generationId"].(float64); ok {
		return fmt.Sprintf("%d", int(genId)), nil
	}
	return "", fmt.Errorf("generationId tidak ditemukan di response. Body: %s", string(body))
}

// SetGenerationID menyimpan generationId terbaru secara thread-safe.
func (c *Client) SetGenerationID(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config.GenerationID = id
}

// GetCachedGenerationID mengembalikan generationId yang tersimpan secara thread-safe.
func (c *Client) GetCachedGenerationID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config.GenerationID
}

// RefreshGenerationID mengambil generationId terbaru dan menyimpannya.
func (c *Client) RefreshGenerationID(reportId string, params string) error {
	id, err := c.GetLatestGenerationID(reportId, params)
	if err != nil {
		return err
	}
	c.SetGenerationID(id)
	log.Println("[INFO] AD Manager Plus - Refreshed Generation ID:", id)
	return nil
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

// FetchComputerReport mengambil data laporan menggunakan generationId yang tersimpan.
func (c *Client) FetchComputerReport() ([]byte, error) {
	// Hardcoded reportId untuk 'All Computers' (bisa diubah jika perlu)
	reportId := "210"
	// Params sesuai contoh, bisa diubah/dibuat dinamis jika perlu
	params := `{"selectedDomains":["satnusa.com"],"domainVsOUList":{"DC=satnusa,DC=com":[]},"domainVsExcludeOUList":{"DC=satnusa,DC=com":[]},"domainVsExcludeChildOU":{"DC=satnusa,DC=com":false}}`
	generationId := c.GetCachedGenerationID()
	if generationId == "" {
		id, err := c.GetLatestGenerationID(reportId, params)
		if err != nil {
			return nil, fmt.Errorf("gagal mendapatkan generationId: %v", err)
		}
		c.SetGenerationID(id)
		generationId = id
	}
	log.Println("[INFO] AD Manager Plus - Using Generation ID:", generationId)
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
	reportPayload.Set("reportId", reportId)
	reportPayload.Set("generationId", generationId)
	reportPayload.Set("params", string(paramsJSON))
	reportPayload.Set("intersect", "false")
	reportPayload.Set("admpcsrf", csrfToken)

	// Polling: try up to 10 times, 2 seconds apart, until valid JSON is returned
	var lastErr error
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("POST", reportURL, strings.NewReader(reportPayload.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("gagal melakukan POST request laporan: %v", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("gagal mengambil laporan, status code: %d, body: %s", resp.StatusCode, string(body))
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		// Try to unmarshal to map[string]interface{} to check if valid JSON
		var test map[string]interface{}
		if err := json.Unmarshal(body, &test); err == nil && len(body) > 0 {
			return body, nil
		}
		lastErr = fmt.Errorf("response belum valid JSON, percobaan ke-%d", i+1)
		// Wait 2 seconds before next try
		time.Sleep(2 * time.Second)
	}
	return nil, lastErr
}
