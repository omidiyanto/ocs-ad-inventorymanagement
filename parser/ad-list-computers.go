package parser

import (
	"strconv"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// ComputerReportRow adalah struktur untuk hasil akhir yang kita inginkan.
// Stuktur ini tetap sama agar kompatibel dengan fungsi Combine.
type ComputerReportRow struct {
	ComputerName     string `json:"computer_name"`
	LastLogonTime    string `json:"last_logon_time"`
	ComputerStatus   string `json:"computer_status"`
	LastModifiedTime string `json:"ad_last_modified_time"`
}

// convertLDAPTimestamp mengonversi Windows NT FileTime (dalam bentuk string) ke format "YYYY-MM-DD HH:MM:SS" dalam zona waktu UTC+7.
func convertLDAPTimestamp(ts string) string {
	if ts == "" || ts == "0" {
		return "0"
	}
	i, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return ""
	}

	unixTimestamp := (i / 10000000) - 11644473600
	t := time.Unix(unixTimestamp, 0).UTC()

	// Tambahkan 7 jam untuk konversi ke UTC+7 (WIB)
	tInWIB := t.Add(7 * time.Hour)

	return tInWIB.Format("2006-01-02 15:04:05")
}

// parseGeneralizedTime mengonversi format waktu LDAP "generalized" ke format yang kita inginkan dalam zona waktu UTC+7.
func parseGeneralizedTime(gt string) string {
	if gt == "" {
		return ""
	}
	// Format LDAP: 20230922083015.0Z
	t, err := time.Parse("20060102150405.0Z", gt)
	if err != nil {
		return ""
	}

	// Tambahkan 7 jam untuk konversi ke UTC+7 (WIB)
	tInWIB := t.Add(7 * time.Hour)

	return tInWIB.Format("2006-01-02 15:04:05")
}

// ParseComputerReportFromLDAP mengubah data mentah LDAP menjadi format sederhana.
func ParseComputerReportFromLDAP(entries []*ldap.Entry) ([]ComputerReportRow, error) {
	var simplifiedList []ComputerReportRow
	const ufAccountDisable = 2

	for _, entry := range entries {
		name := entry.GetAttributeValue("name")
		if name == "" {
			continue // Skip entri tanpa nama
		}

		uacStr := entry.GetAttributeValue("userAccountControl")
		uac, _ := strconv.Atoi(uacStr)
		status := "enabled"
		// Cek flag ACCOUNTDISABLE
		if (uac & ufAccountDisable) == ufAccountDisable {
			status = "disabled"
		}

		simplifiedList = append(simplifiedList, ComputerReportRow{
			ComputerName:     name,
			LastLogonTime:    convertLDAPTimestamp(entry.GetAttributeValue("lastLogonTimestamp")),
			ComputerStatus:   strings.ToLower(status),
			LastModifiedTime: parseGeneralizedTime(entry.GetAttributeValue("whenChanged")),
		})
	}

	return simplifiedList, nil
}
