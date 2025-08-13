package parser

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

type FinalComputerRow struct {
	ComputerName     string `json:"computer_name"`
	ExistsInOCS      bool   `json:"exists_in_ocs"`
	ExistsInAD       bool   `json:"exists_in_ad"`
	OCSStatus        string `json:"ocs_status"`
	ADStatus         string `json:"ad_status"`
	OCSLastInventory string `json:"ocs_last_inventory"`
	OCSLastCome      string `json:"ocs_last_come"`
	ADLastLogonTime  string `json:"ad_last_logon_time"`
}

// Deduplication dan penggabungan data OCS dan AD
func CombineOCSAndAD(ocsList []OCSComputerRow, adList []ComputerReportRow) []FinalComputerRow {
	cache := make(map[string]string) // hash -> computer_name
	result := make(map[string]*FinalComputerRow)

	// Helper hashing
	hashName := func(name string) string {
		h := sha1.New()
		h.Write([]byte(strings.ToLower(strings.TrimSpace(name))))
		return hex.EncodeToString(h.Sum(nil))
	}

	// Masukkan OCS
	for _, ocs := range ocsList {
		key := hashName(ocs.ComputerName)
		cache[key] = ocs.ComputerName
		result[ocs.ComputerName] = &FinalComputerRow{
			ComputerName:     ocs.ComputerName,
			ExistsInOCS:      true,
			ExistsInAD:       false,
			OCSStatus:        ocs.OCSStatus,
			ADStatus:         "",
			OCSLastInventory: ocs.OCSLastInventory,
			OCSLastCome:      ocs.OCSLastCome,
			ADLastLogonTime:  "",
		}
	}

	// Masukkan AD
	for _, ad := range adList {
		key := hashName(ad.ComputerName)
		if cname, ok := cache[key]; ok {
			// Sudah ada di OCS
			row := result[cname]
			row.ExistsInAD = true
			row.ADStatus = ad.ComputerStatus
			row.ADLastLogonTime = ad.LastLogonTime
		} else {
			// Hanya di AD
			result[ad.ComputerName] = &FinalComputerRow{
				ComputerName:     ad.ComputerName,
				ExistsInOCS:      false,
				ExistsInAD:       true,
				OCSStatus:        "",
				ADStatus:         ad.ComputerStatus,
				OCSLastInventory: "",
				OCSLastCome:      "",
				ADLastLogonTime:  ad.LastLogonTime,
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
