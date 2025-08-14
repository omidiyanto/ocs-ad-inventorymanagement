package parser

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"
)

type FinalComputerRow struct {
	ComputerName          string  `json:"computer_name"`
	ExistsInOCS           bool    `json:"exists_in_ocs"`
	ExistsInAD            bool    `json:"exists_in_ad"`
	OCSStatus             string  `json:"ocs_status"`
	ADStatus              string  `json:"ad_status"`
	OCSLastInventory      *string `json:"ocs_last_inventory,omitempty"`
	OCSLastCome           *string `json:"ocs_last_come,omitempty"`
	ADLastLogonTime       *string `json:"ad_last_logon_time,omitempty"`
	ADNotLoginMoreThan30d bool    `json:"ad_not_login_more_than_30d"`
	ADNotLoginMoreThan45d bool    `json:"ad_not_login_more_than_45d"`
}

// Deduplication dan penggabungan data OCS dan AD
func HashComputerName(name string) string {
	h := sha1.New()
	h.Write([]byte(strings.ToLower(strings.TrimSpace(name))))
	return hex.EncodeToString(h.Sum(nil))
}

// Deduplication dan penggabungan data OCS dan AD
func CombineOCSAndAD(ocsList []OCSComputerRow, adList []ComputerReportRow) []FinalComputerRow {
	cache := make(map[string]string) // hash -> computer_name
	result := make(map[string]*FinalComputerRow)

	// Helper untuk cek login > 30/45 hari
	parseTime := func(t string) (time.Time, bool) {
		if t == "" {
			return time.Time{}, false
		}
		layouts := []string{"2006-01-02 15:04:05", time.RFC3339, "2006-01-02T15:04:05Z"}
		for _, layout := range layouts {
			parsed, err := time.Parse(layout, t)
			if err == nil {
				return parsed, true
			}
		}
		return time.Time{}, false
	}
	now := time.Now()

	// Helper konversi ke RFC3339
	toRFC3339Ptr := func(t string) *string {
		if t == "" {
			return nil
		}
		layouts := []string{"2006-01-02 15:04:05", time.RFC3339, "2006-01-02T15:04:05Z"}
		for _, layout := range layouts {
			parsed, err := time.Parse(layout, t)
			if err == nil {
				s := parsed.UTC().Format(time.RFC3339)
				return &s
			}
		}
		return nil
	}

	// Masukkan OCS
	for _, ocs := range ocsList {
		key := HashComputerName(ocs.ComputerName)
		cache[key] = ocs.ComputerName
		result[ocs.ComputerName] = &FinalComputerRow{
			ComputerName:          ocs.ComputerName,
			ExistsInOCS:           true,
			ExistsInAD:            false,
			OCSStatus:             ocs.OCSStatus,
			ADStatus:              "",
			OCSLastInventory:      toRFC3339Ptr(ocs.OCSLastInventory),
			OCSLastCome:           toRFC3339Ptr(ocs.OCSLastCome),
			ADLastLogonTime:       nil,
			ADNotLoginMoreThan30d: false,
			ADNotLoginMoreThan45d: false,
		}
	}

	// Masukkan AD
	for _, ad := range adList {
		key := HashComputerName(ad.ComputerName)
		// Hitung field login > 30/45 hari
		moreThan30d, moreThan45d := false, false
		if t, ok := parseTime(ad.LastLogonTime); ok {
			diff := now.Sub(t)
			if diff.Hours() > 24*30 {
				moreThan30d = true
			}
			if diff.Hours() > 24*45 {
				moreThan45d = true
			}
		}
		if cname, ok := cache[key]; ok {
			// Sudah ada di OCS
			row := result[cname]
			row.ExistsInAD = true
			row.ADStatus = ad.ComputerStatus
			row.ADLastLogonTime = toRFC3339Ptr(ad.LastLogonTime)
			row.ADNotLoginMoreThan30d = moreThan30d
			row.ADNotLoginMoreThan45d = moreThan45d
		} else {
			// Hanya di AD
			result[ad.ComputerName] = &FinalComputerRow{
				ComputerName:          ad.ComputerName,
				ExistsInOCS:           false,
				ExistsInAD:            true,
				OCSStatus:             "",
				ADStatus:              ad.ComputerStatus,
				OCSLastInventory:      nil,
				OCSLastCome:           nil,
				ADLastLogonTime:       toRFC3339Ptr(ad.LastLogonTime),
				ADNotLoginMoreThan30d: moreThan30d,
				ADNotLoginMoreThan45d: moreThan45d,
			}
		}
	}

	// Convert ke slice
	var finalList []FinalComputerRow
	for _, v := range result {
		finalList = append(finalList, *v)
	}
	return finalList
}
