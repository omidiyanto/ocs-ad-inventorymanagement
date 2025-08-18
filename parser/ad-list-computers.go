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
	// fmt.Println("\n[DEBUG] RAW JSON FROM AD:")
	// fmt.Println(string(rawData))
	var simplifiedList []ComputerReportRow

	idToKeyMap := map[int]string{
		3001: "computer_name",
		3019: "last_logon_time",
		3021: "computer_status",
		3012: "ad_last_modified_time",
	}

	// Try to unmarshal as the usual format first
	var parsedData RawReportData
	err := json.Unmarshal(rawData, &parsedData)
	if err == nil && len(parsedData.ResultRows) > 0 {
		for _, row := range parsedData.ResultRows {
			item := make(map[string]string)
			for _, col := range row.Columns {
				if key, ok := idToKeyMap[col.AttribID]; ok {
					item[key] = col.Value
				}
			}
			if name, ok := item["computer_name"]; ok {
				logon := item["last_logon_time"]
				status := item["computer_status"]
				modified := item["ad_last_modified_time"]
				simplifiedList = append(simplifiedList, ComputerReportRow{
					ComputerName:     name,
					LastLogonTime:    logon,
					ComputerStatus:   strings.ToLower(status),
					LastModifiedTime: modified,
				})
			}
		}
		return simplifiedList, nil
	}

	// If not, try to parse as array of objects keyed by ATTRIB_ID
	// The root may be an object with "resultrows" as []map[string]interface{}
	var generic map[string]interface{}
	if err := json.Unmarshal(rawData, &generic); err != nil {
		return nil, fmt.Errorf("gagal mem-parsing JSON mentah: %v", err)
	}
	rows, ok := generic["resultrows"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("format resultrows tidak dikenali")
	}
	for _, r := range rows {
		item := make(map[string]string)
		// r bisa berupa map[string]interface{} dengan key ATTRIB_ID
		if rowMap, ok := r.(map[string]interface{}); ok {
			for k, v := range rowMap {
				// k is always string, representing ATTRIB_ID
				var id int
				fmt.Sscanf(k, "%d", &id)
				if key, ok := idToKeyMap[id]; ok {
					switch val := v.(type) {
					case string:
						item[key] = val
					case float64:
						item[key] = fmt.Sprintf("%v", val)
					case int:
						item[key] = fmt.Sprintf("%v", val)
					}
				}
			}
			if name, ok := item["computer_name"]; ok {
				logon := item["last_logon_time"]
				status := item["computer_status"]
				modified := item["ad_last_modified_time"]
				simplifiedList = append(simplifiedList, ComputerReportRow{
					ComputerName:     name,
					LastLogonTime:    logon,
					ComputerStatus:   strings.ToLower(status),
					LastModifiedTime: modified,
				})
			}
		}
	}
	return simplifiedList, nil
}
