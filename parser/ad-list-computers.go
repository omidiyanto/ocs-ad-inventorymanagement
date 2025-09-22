package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// TAMBAHKAN field baru untuk menyimpan informasi OS
type ComputerReportRow struct {
	ComputerName     string `json:"computer_name"`
	OperatingSystem  string `json:"operating_system"`
	LastLogonTime    string `json:"last_logon_time"`
	ComputerStatus   string `json:"computer_status"`
	LastModifiedTime string `json:"ad_last_modified_time"`
}

func convertLDAPTimestamp(ts string) string {
	if ts == "" || ts == "0" {
		return "0"
	}
	i, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || i == 0 {
		return "0"
	}

	unixTimestamp := (i / 10000000) - 11644473600
	t := time.Unix(unixTimestamp, 0).UTC()
	tInWIB := t.Add(7 * time.Hour)
	return tInWIB.Format("2006-01-02 15:04:05")
}

func parseGeneralizedTime(gt string) string {
	if gt == "" {
		return ""
	}
	t, err := time.Parse("20060102150405.0Z", gt)
	if err != nil {
		return ""
	}
	tInWIB := t.Add(7 * time.Hour)
	return tInWIB.Format("2006-01-02 15:04:05")
}

func ParseComputerReportFromLDAP(entries []*ldap.Entry) ([]ComputerReportRow, error) {
	var simplifiedList []ComputerReportRow
	const ufAccountDisable = 2

	for _, entry := range entries {
		name := entry.GetAttributeValue("name")
		if name == "" {
			continue
		}

		// AMBIL data OS dan format
		os := entry.GetAttributeValue("operatingSystem")
		osVersion := entry.GetAttributeValue("operatingSystemVersion")
		fullOS := os
		if osVersion != "" {
			fullOS = fmt.Sprintf("%s (%s)", os, osVersion)
		}

		uacStr := entry.GetAttributeValue("userAccountControl")
		uac, _ := strconv.Atoi(uacStr)
		status := "enabled"
		if (uac & ufAccountDisable) == ufAccountDisable {
			status = "disabled"
		}

		// Prioritaskan lastLogon (real-time), jika kosong baru pakai lastLogonTimestamp
		lastLogon := convertLDAPTimestamp(entry.GetAttributeValue("lastLogon"))
		if lastLogon == "0" {
			lastLogon = convertLDAPTimestamp(entry.GetAttributeValue("lastLogonTimestamp"))
		}

		simplifiedList = append(simplifiedList, ComputerReportRow{
			ComputerName:     name,
			OperatingSystem:  fullOS, // Simpan informasi OS
			LastLogonTime:    lastLogon,
			ComputerStatus:   strings.ToLower(status),
			LastModifiedTime: parseGeneralizedTime(entry.GetAttributeValue("whenChanged")),
		})
	}

	return simplifiedList, nil
}
