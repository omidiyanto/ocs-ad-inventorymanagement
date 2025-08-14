package api

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeleteComputerHandler handles GET /delete-computer?name=xxx
// Versi ini tetap menggunakan introspeksi skema namun dengan eksekusi query yang lebih aman.
func DeleteComputerHandler(db *gorm.DB) gin.HandlerFunc {
	// Regex untuk validasi nama tabel tetap dipertahankan sebagai lapisan pertahanan tambahan (defense-in-depth).
	var validTableName = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

	// Whitelist nama tabel yang diizinkan
	var allowedTables = map[string]struct{}{
		"accesslog": {}, "accountinfo": {}, "accountinfo_config": {}, "archive": {}, "assets_categories": {}, "auth_attempt": {}, "batteries": {}, "bios": {}, "blacklist_macaddresses": {}, "blacklist_serials": {}, "blacklist_subnet": {}, "config": {}, "config_ldap": {}, "conntrack": {}, "controllers": {}, "cpus": {}, "cve_search": {}, "cve_search_computer": {}, "cve_search_correspondance": {}, "cve_search_history": {}, "deleted_equiv": {}, "deploy": {}, "devices": {}, "devicetype": {}, "dico_ignored": {}, "dico_soft": {}, "download_affect_rules": {}, "download_available": {}, "download_enable": {}, "download_history": {}, "download_servers": {}, "downloadwk_conf_values": {}, "downloadwk_fields": {}, "downloadwk_history": {}, "downloadwk_pack": {}, "downloadwk_statut_request": {}, "downloadwk_tab_values": {}, "drives": {}, "engine_mutex": {}, "engine_persistent": {}, "extensions": {}, "files": {}, "groups": {}, "groups_cache": {}, "hardware": {}, "hardware_osname_cache": {}, "history": {}, "inputs": {}, "itmgmt_comments": {}, "javainfo": {}, "journallog": {}, "languages": {}, "layouts": {}, "local_groups": {}, "local_users": {}, "locks": {}, "memories": {}, "modems": {}, "monitors": {}, "netmap": {}, "network_devices": {}, "networks": {}, "notification": {}, "notification_config": {}, "ports": {}, "printers": {}, "prolog_conntrack": {}, "regconfig": {}, "registry": {}, "registry_name_cache": {}, "registry_regvalue_cache": {}, "reports_notifications": {}, "repository": {}, "saas": {}, "saas_exp": {}, "save_query": {}, "schedule_wol": {}, "sim": {}, "slots": {}, "snmp_accountinfo": {}, "snmp_communities": {}, "snmp_configs": {}, "snmp_default": {}, "snmp_labels": {}, "snmp_mibs": {}, "snmp_ocs": {}, "snmp_types": {}, "snmp_types_conditions": {}, "software": {}, "software_categories": {}, "software_categories_link": {}, "software_category_exp": {}, "software_link": {}, "software_name": {}, "software_publisher": {}, "software_version": {}, "softwares_name_cache": {}, "sounds": {}, "ssl_store": {}, "storages": {}, "subnet": {}, "tags": {}, "temp_files": {}, "usbdevices": {}, "videos": {}, "virtualmachines": {},
	}

	return func(c *gin.Context) {
		name := c.Query("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parameter 'name' wajib diisi"})
			return
		}

		// Struct sementara untuk menampung hasil query ID
		var hardware struct {
			ID int
		}

		// Cari id hardware berdasarkan nama.
		if err := db.Table("hardware").Select("id").Where("name = ?", name).First(&hardware).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Computer tidak ditemukan"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Gagal mencari hardware: %v", err)})
			return
		}

		hwID := hardware.ID

		// Ambil daftar tabel yang punya kolom HARDWARE_ID di skema saat ini (logika ini dipertahankan sesuai permintaan).
		type tableRow struct {
			TableName string `gorm:"column:TABLE_NAME"`
		}
		var tables []tableRow
		query := `
			SELECT DISTINCT TABLE_NAME
			FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE() AND COLUMN_NAME = 'HARDWARE_ID'
		`
		if err := db.Raw(query).Scan(&tables).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Gagal mengambil daftar tabel: %v", err)})
			return
		}

		// Mulai transaksi
		tx := db.Begin()
		if tx.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Gagal memulai transaksi: %v", tx.Error)})
			return
		}

		// Defer a rollback in case of panic
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		for _, t := range tables {
			tableName := t.TableName

			// Lakukan validasi nama tabel sebagai lapisan keamanan tambahan.
			if !validTableName.MatchString(tableName) {
				// skip tabel yang namanya tidak valid untuk mencegah hal tak terduga.
				continue
			}
			// Validasi whitelist nama tabel
			if _, ok := allowedTables[tableName]; !ok {
				// skip tabel yang tidak ada di whitelist
				continue
			}

			// --- PERUBAHAN UTAMA ADA DI SINI ---
			// Ganti Sprintf dengan metode GORM yang aman untuk nama tabel dinamis.
			// GORM akan menangani quoting (misal: `nama_tabel`) secara otomatis dan aman.
			if err := tx.Table(tableName).Where("HARDWARE_ID = ?", hwID).Delete(nil).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Gagal menghapus dari tabel %s: %v", tableName, err)})
				return
			}
		}

		// Hapus record hardware itu sendiri
		if err := tx.Table("hardware").Where("id = ?", hwID).Delete(nil).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Gagal menghapus hardware: %v", err)})
			return
		}

		// Commit transaksi
		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Gagal commit transaksi: %v", err)})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Semua data yang terkait dengan computer ID %d telah berhasil dihapus.", hwID),
		})
	}
}
