package parser

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"
)

type FinalComputerRow struct {
	ComputerName                string `json:"computer_name"`
	ExistsInOCS                 bool   `json:"exists_in_ocs"`
	ExistsInAD                  bool   `json:"exists_in_ad"`
	OCSStatus                   string `json:"ocs_status"`
	ADStatus                    string `json:"ad_status"`
	OCSLastInventory            string `json:"ocs_last_inventory,omitempty"`
	OCSLastCome                 string `json:"ocs_last_come,omitempty"`
	ADLastLogonTime             string `json:"ad_last_logon_time,omitempty"`
	ADNotLoginMoreThan30d       *bool  `json:"ad_not_login_more_than_30d,omitempty"`
	ADNotLoginMoreThan45d       *bool  `json:"ad_not_login_more_than_45d,omitempty"`
	OCSLastInventoryMoreThan30d *bool  `json:"ocs_last_inventory_more_than_30d,omitempty"`
	OCSLastComeMoreThan30d      *bool  `json:"ocs_last_come_more_than_30d,omitempty"`
	OCSInactiveDurationDays     *int   `json:"ocs_inactive_duration_days,omitempty"`
	ADInactiveDurationDays      *int   `json:"ad_inactive_duration_days,omitempty"`
	SyncTime                    string `json:"@timestamp"`
}

// HashComputerName membuat hash dari nama komputer untuk deduplikasi.
func HashComputerName(name string) string {
	h := sha1.New()
	h.Write([]byte(strings.ToLower(strings.TrimSpace(name))))
	return hex.EncodeToString(h.Sum(nil))
}

// CombineOCSAndAD melakukan deduplikasi dan penggabungan data dari OCS dan AD.
func CombineOCSAndAD(ocsList []OCSComputerRow, adList []ComputerReportRow) []FinalComputerRow {
	cache := make(map[string]string) // hash -> computer_name
	result := make(map[string]*FinalComputerRow)

	// Helper untuk mem-parsing berbagai format waktu, mengembalikan time.Time dan status keberhasilan.
	parseTime := func(t string) (time.Time, bool) {
		if t == "" {
			return time.Time{}, false
		}
		// Daftar layout format waktu yang mungkin diterima
		layouts := []string{"2006-01-02 15:04:05", time.RFC3339, "2006-01-02T15:04:05Z"}
		for _, layout := range layouts {
			parsed, err := time.Parse(layout, t)
			if err == nil {
				return parsed, true
			}
		}
		return time.Time{}, false
	}

	// Helper baru untuk mengisi @timestamp (SyncTime) dengan zona waktu UTC-7.
	// Fungsi ini memilih timestamp prioritas (OCS > AD), memformatnya ke RFC3339,
	// dan memberikan nilai default jika keduanya tidak valid.
	getSyncTimestamp := func(ocsLastInventory, adLastLogon string) string {
		shift := -7 * time.Hour
		isZeroOrDash := func(s string) bool {
			return s == "0" || s == "-"
		}
		// Prioritas 1: Coba parse OCSLastInventory
		if !isZeroOrDash(ocsLastInventory) {
			if t, ok := parseTime(ocsLastInventory); ok {
				return t.Add(shift).UTC().Format(time.RFC3339)
			}
		}
		// Prioritas 2: Coba parse ADLastLogon
		if !isZeroOrDash(adLastLogon) {
			if t, ok := parseTime(adLastLogon); ok {
				return t.Add(shift).UTC().Format(time.RFC3339)
			}
		}
		// Jika field bernilai 0 atau -, gunakan waktu UTC+0 (tanpa shift)
		return time.Now().UTC().Format(time.RFC3339)
	}

	// Helper untuk pengecekan string "0" atau "-" dan waktu sekarang
	isZeroOrDash := func(s string) bool {
		return s == "0" || s == "-"
	}
	now := time.Now().Local()

	// Proses data dari OCS
	for _, ocs := range ocsList {
		key := HashComputerName(ocs.ComputerName)
		cache[key] = ocs.ComputerName

		// Logic untuk ocs_last_inventory_more_than_30d dan ocs_last_come_more_than_30d
		var ocsLastInventoryMoreThan30d *bool
		var ocsLastComeMoreThan30d *bool
		var ocsInactiveDurationDays *int
		if !isZeroOrDash(ocs.OCSLastInventory) {
			if t, ok := parseTime(ocs.OCSLastInventory); ok {
				b := now.Sub(t).Hours() > 24*30
				ocsLastInventoryMoreThan30d = &b
			}
		} else {
			b := true
			ocsLastInventoryMoreThan30d = &b
		}
		if !isZeroOrDash(ocs.OCSLastCome) {
			if t, ok := parseTime(ocs.OCSLastCome); ok {
				b := now.Sub(t).Hours() > 24*30
				ocsLastComeMoreThan30d = &b
				days := int(now.Sub(t).Hours() / 24)
				ocsInactiveDurationDays = &days
			}
		} else {
			b := true
			ocsLastComeMoreThan30d = &b
		}

		result[ocs.ComputerName] = &FinalComputerRow{
			ComputerName: ocs.ComputerName,
			ExistsInOCS:  true,
			ExistsInAD:   false,
			OCSStatus:    ocs.OCSStatus,
			ADStatus:     "",
			// Aturan 2: Biarkan format string original
			OCSLastInventory:            ocs.OCSLastInventory,
			OCSLastCome:                 ocs.OCSLastCome,
			ADLastLogonTime:             "",
			ADNotLoginMoreThan30d:       nil,
			ADNotLoginMoreThan45d:       nil,
			OCSLastInventoryMoreThan30d: ocsLastInventoryMoreThan30d,
			OCSLastComeMoreThan30d:      ocsLastComeMoreThan30d,
			OCSInactiveDurationDays:     ocsInactiveDurationDays,
			ADInactiveDurationDays:      nil,
			// Aturan 1 & 3: Buat @timestamp dalam format RFC3339 dari data OCS
			SyncTime: getSyncTimestamp(ocs.OCSLastInventory, ""),
		}
	}

	// Proses data dari AD dan gabungkan dengan data OCS yang ada
	// now dan isZeroOrDash sudah didefinisikan di atas
	for _, ad := range adList {
		key := HashComputerName(ad.ComputerName)

		// Hitung field login > 30/45 hari
		var moreThan30d *bool
		var moreThan45d *bool
		var adInactiveDurationDays *int
		if !isZeroOrDash(ad.LastLogonTime) {
			if t, ok := parseTime(ad.LastLogonTime); ok {
				b30 := now.Sub(t).Hours() > 24*30
				b45 := now.Sub(t).Hours() > 24*45
				moreThan30d = &b30
				moreThan45d = &b45
				days := int(now.Sub(t).Hours() / 24)
				adInactiveDurationDays = &days
			}
		} else {
			b := true
			moreThan30d = &b
			moreThan45d = &b
		}

		if cname, ok := cache[key]; ok {
			// Komputer sudah ada di OCS, update data AD
			row := result[cname]
			row.ExistsInAD = true
			row.ADStatus = ad.ComputerStatus
			// Aturan 2: Biarkan format string original
			row.ADLastLogonTime = ad.LastLogonTime
			row.ADNotLoginMoreThan30d = moreThan30d
			row.ADNotLoginMoreThan45d = moreThan45d
			row.ADInactiveDurationDays = adInactiveDurationDays
			// Aturan 1 & 3: Update @timestamp dengan mempertimbangkan data OCS dan AD
			row.SyncTime = getSyncTimestamp(row.OCSLastInventory, ad.LastLogonTime)
		} else {
			// Komputer hanya ada di AD
			result[ad.ComputerName] = &FinalComputerRow{
				ComputerName:     ad.ComputerName,
				ExistsInOCS:      false,
				ExistsInAD:       true,
				OCSStatus:        "",
				ADStatus:         ad.ComputerStatus,
				OCSLastInventory: "",
				OCSLastCome:      "",
				// Aturan 2: Biarkan format string original
				ADLastLogonTime:             ad.LastLogonTime,
				ADNotLoginMoreThan30d:       moreThan30d,
				ADNotLoginMoreThan45d:       moreThan45d,
				OCSLastInventoryMoreThan30d: nil,
				OCSLastComeMoreThan30d:      nil,
				OCSInactiveDurationDays:     nil,
				ADInactiveDurationDays:      adInactiveDurationDays,
				// Aturan 1 & 3: Buat @timestamp dalam format RFC3339 dari data AD
				SyncTime: getSyncTimestamp("", ad.LastLogonTime),
			}
		}
	}

	// Konversi map hasil ke slice untuk dikembalikan
	var finalList []FinalComputerRow
	for _, v := range result {
		finalList = append(finalList, *v)
	}
	return finalList
}
