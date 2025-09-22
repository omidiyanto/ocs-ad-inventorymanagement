package client

import (
	"fmt"
	"os"

	"github.com/go-ldap/ldap/v3"
)

// LDAPConfig menyimpan konfigurasi koneksi ke server LDAP (Active Directory)
type LDAPConfig struct {
	Host       string
	Port       int
	BindDN     string
	BindPass   string
	SearchBase string
}

// LoadLDAPConfig memuat konfigurasi LDAP dari environment variables
func LoadLDAPConfig() LDAPConfig {
	portStr := os.Getenv("LDAP_PORT")
	if portStr == "" {
		portStr = "389" // Default LDAP port
	}
	port, _ := Atoi(portStr)

	return LDAPConfig{
		Host:       os.Getenv("LDAP_HOST"),
		Port:       port,
		BindDN:     os.Getenv("LDAP_BIND_DN"),
		BindPass:   os.Getenv("LDAP_BIND_PASSWORD"),
		SearchBase: os.Getenv("LDAP_SEARCH_BASE"),
	}
}

// LDAPClient adalah client untuk koneksi LDAP
type LDAPClient struct {
	Conn   *ldap.Conn
	Config LDAPConfig
}

// NewLDAPClient membuat client baru dan melakukan koneksi serta bind ke server LDAP
func NewLDAPClient(cfg LDAPConfig) (*LDAPClient, error) {
	conn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("gagal koneksi ke server LDAP: %v", err)
	}

	err = conn.Bind(cfg.BindDN, cfg.BindPass)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("gagal bind/autentikasi ke LDAP: %v", err)
	}

	return &LDAPClient{Conn: conn, Config: cfg}, nil
}

// ListComputers mengambil semua objek komputer dari Active Directory
func (c *LDAPClient) ListComputers() ([]*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		c.Config.SearchBase,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=computer)",
		// TAMBAHKAN atribut operatingSystem dan operatingSystemVersion
		[]string{"name", "operatingSystem", "operatingSystemVersion", "lastLogon", "lastLogonTimestamp", "whenChanged", "userAccountControl"},
		nil,
	)

	sr, err := c.Conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("pencarian LDAP gagal: %v", err)
	}

	return sr.Entries, nil
}

// Close menutup koneksi LDAP
func (c *LDAPClient) Close() {
	if c.Conn != nil {
		c.Conn.Close()
	}
}

func Atoi(s string) (int, error) {
	var i int
	_, err := fmt.Sscan(s, &i)
	return i, err
}
