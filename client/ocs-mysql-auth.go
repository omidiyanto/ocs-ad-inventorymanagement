// client/ocs-mysql-auth.go
package client

import (
	"fmt"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// OCSConfig menyimpan konfigurasi koneksi OCS MySQL
type OCSConfig struct {
	DBUrl  string
	DBPort string
	DBName string
	DBUser string
	DBPass string
}

// LoadOCSConfig membaca konfigurasi dari environment
func LoadOCSConfig() OCSConfig {
	return OCSConfig{
		DBUrl:  os.Getenv("OCS_DB_URL"),
		DBPort: os.Getenv("OCS_DB_PORT"),
		DBName: os.Getenv("OCS_DB_NAME"),
		DBUser: os.Getenv("OCS_DB_USER"),
		DBPass: os.Getenv("OCS_DB_PASS"),
	}
}

// OCSMySQLClient adalah client untuk koneksi OCS MySQL
type OCSMySQLClient struct {
	DB     *gorm.DB
	Config OCSConfig
}

// NewOCSMySQLClient membuat client baru dan mencoba koneksi ke OCS MySQL
func NewOCSMySQLClient(cfg OCSConfig) (*OCSMySQLClient, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPass, cfg.DBUrl, cfg.DBPort, cfg.DBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &OCSMySQLClient{DB: db, Config: cfg}, nil
}

// Ping untuk cek koneksi
func (c *OCSMySQLClient) Ping() error {
	db, err := c.DB.DB()
	if err != nil {
		return err
	}
	return db.Ping()
}
