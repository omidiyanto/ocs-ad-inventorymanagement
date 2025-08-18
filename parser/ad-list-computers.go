// parser/ad-list-computers.go
package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RawReportColumn mendefinisikan struktur satu kolom dalam data mentah.
type RawReportColumn struct {
	Value    string `json:"VALUE"`
	AttribID int    `json:"ATTRIB_ID"`
}

// RawReportRow mendefinisikan struktur satu baris dalam data mentah.
type RawReportRow struct {
	Columns []RawReportColumn `json:"COLUMNS"`
}

// RawReportData adalah struktur tingkat atas dari JSON mentah.
type RawReportData struct {
	ResultRows []RawReportRow `json:"resultrows"`
}

// ComputerReportRow adalah struktur untuk hasil akhir yang kita inginkan.
type ComputerReportRow struct {
	ComputerName     string `json:"computer_name"`
	LastLogonTime    string `json:"last_logon_time"`
	ComputerStatus   string `json:"computer_status"`
	LastModifiedTime string `json:"ad_last_modified_time"`
}

// ParseComputerReport mengubah data mentah menjadi format sederhana.
func ParseComputerReport(rawData []byte) ([]ComputerReportRow, error) {
	// fmt.Println("\n[*] Memulai transformasi data JSON...")
	var parsedData RawReportData
	if err := json.Unmarshal(rawData, &parsedData); err != nil {
		return nil, fmt.Errorf("gagal mem-parsing JSON mentah: %v", err)
	}

	var simplifiedList []ComputerReportRow

	idToKeyMap := map[int]string{
		3001: "computer_name",
		3019: "last_logon_time",
		3021: "computer_status",
		3012: "ad_last_modified_time",
	}

	for _, row := range parsedData.ResultRows {
		item := make(map[string]string)
		for _, col := range row.Columns {
			if key, ok := idToKeyMap[col.AttribID]; ok {
				item[key] = col.Value
			}
		}

		if name, ok := item["computer_name"]; ok {
			logon, _ := item["last_logon_time"]
			status, _ := item["computer_status"]
			modified, _ := item["ad_last_modified_time"]
			simplifiedList = append(simplifiedList, ComputerReportRow{
				ComputerName:     name,
				LastLogonTime:    logon,
				ComputerStatus:   strings.ToLower(status),
				LastModifiedTime: modified,
			})
		}
	}
	// fmt.Println("[+] Transformasi data selesai.")
	return simplifiedList, nil
}
