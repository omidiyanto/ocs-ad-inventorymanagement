package parser

import (
	"time"

	"gorm.io/gorm"
)

type OCSComputerRow struct {
	ComputerName     string `json:"computer_name"`
	OCSStatus        string `json:"ocs_status"`
	OCSLastCome      string `json:"ocs_last_come"`
	OCSLastInventory string `json:"ocs_last_inventory"`
}

type Hardware struct {
	Name     string     `gorm:"column:NAME"`
	Archive  *int       `gorm:"column:ARCHIVE"`
	LastDate *time.Time `gorm:"column:LASTDATE"`
	LastCome *time.Time `gorm:"column:LASTCOME"`
}

func (Hardware) TableName() string {
	return "hardware"
}

func ListOCSComputers(db *gorm.DB, limit int) ([]OCSComputerRow, error) {
	var hardwares []Hardware
	q := db
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&hardwares).Error; err != nil {
		return nil, err
	}
	var result []OCSComputerRow
	for _, hw := range hardwares {
		status := "enabled"
		if hw.Archive != nil {
			status = "disabled"
		}
		lastCome := ""
		lastInventory := ""
		if hw.LastCome != nil {
			lastCome = hw.LastCome.Format("2006-01-02 15:04:05")
		}
		if hw.LastDate != nil {
			lastInventory = hw.LastDate.Format("2006-01-02 15:04:05")
		}
		result = append(result, OCSComputerRow{
			ComputerName:     hw.Name,
			OCSStatus:        status,
			OCSLastCome:      lastCome,
			OCSLastInventory: lastInventory,
		})
	}
	return result, nil
}
