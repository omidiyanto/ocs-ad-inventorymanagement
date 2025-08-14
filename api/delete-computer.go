package api

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// DeleteComputerHandler handles GET /delete-computer?name=xxx
// Versi ini tetap menggunakan introspeksi skema namun dengan eksekusi query yang lebih aman.
// Serve frontend if not API call, else process API
func DeleteComputerHandler(db *gorm.DB) gin.HandlerFunc {
	// Regex untuk validasi nama tabel tetap dipertahankan sebagai lapisan pertahanan tambahan (defense-in-depth).
	var validTableName = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

	// Whitelist nama tabel yang diizinkan
	var allowedTables = map[string]struct{}{
		"accesslog": {}, "accountinfo": {}, "accountinfo_config": {}, "archive": {}, "assets_categories": {}, "auth_attempt": {}, "batteries": {}, "bios": {}, "blacklist_macaddresses": {}, "blacklist_serials": {}, "blacklist_subnet": {}, "config": {}, "config_ldap": {}, "conntrack": {}, "controllers": {}, "cpus": {}, "cve_search": {}, "cve_search_computer": {}, "cve_search_correspondance": {}, "cve_search_history": {}, "deleted_equiv": {}, "deploy": {}, "devices": {}, "devicetype": {}, "dico_ignored": {}, "dico_soft": {}, "download_affect_rules": {}, "download_available": {}, "download_enable": {}, "download_history": {}, "download_servers": {}, "downloadwk_conf_values": {}, "downloadwk_fields": {}, "downloadwk_history": {}, "downloadwk_pack": {}, "downloadwk_statut_request": {}, "downloadwk_tab_values": {}, "drives": {}, "engine_mutex": {}, "engine_persistent": {}, "extensions": {}, "files": {}, "groups": {}, "groups_cache": {}, "hardware": {}, "hardware_osname_cache": {}, "history": {}, "inputs": {}, "itmgmt_comments": {}, "javainfo": {}, "journallog": {}, "languages": {}, "layouts": {}, "local_groups": {}, "local_users": {}, "locks": {}, "memories": {}, "modems": {}, "monitors": {}, "netmap": {}, "network_devices": {}, "networks": {}, "notification": {}, "notification_config": {}, "ports": {}, "printers": {}, "prolog_conntrack": {}, "regconfig": {}, "registry": {}, "registry_name_cache": {}, "registry_regvalue_cache": {}, "reports_notifications": {}, "repository": {}, "saas": {}, "saas_exp": {}, "save_query": {}, "schedule_wol": {}, "sim": {}, "slots": {}, "snmp_accountinfo": {}, "snmp_communities": {}, "snmp_configs": {}, "snmp_default": {}, "snmp_labels": {}, "snmp_mibs": {}, "snmp_ocs": {}, "snmp_types": {}, "snmp_types_conditions": {}, "software": {}, "software_categories": {}, "software_categories_link": {}, "software_category_exp": {}, "software_link": {}, "software_name": {}, "software_publisher": {}, "software_version": {}, "softwares_name_cache": {}, "sounds": {}, "ssl_store": {}, "storages": {}, "subnet": {}, "tags": {}, "temp_files": {}, "usbdevices": {}, "videos": {}, "virtualmachines": {},
	}

	return func(c *gin.Context) {
		// If not API call (no Authorization header), serve frontend HTML
		authHeader := c.GetHeader("Authorization")
		name := c.Query("name")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// Serve HTML UI for delete-computer
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, frontendHTML, name)
			return
		}
		// --- JWT Auth ---
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "supersecretjwtkey"
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid"})
			return
		}
		username, _ := claims["username"].(string)
		if username == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid (no username)"})
			return
		}
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
			"message":    fmt.Sprintf("Semua data yang terkait dengan computer ID %d telah berhasil dihapus.", hwID),
			"deleted_by": username,
		})

	}
}

// Embed the frontend HTML as a Go string
var frontendHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Delete Computer - OCS Inventory</title>
	<script src="https://cdn.tailwindcss.com"></script>
	<style>
		body { background: #e6e6e6; }
		.ocs-purple { background: #93318e; color: #fff; }
		.ocs-purple-text { color: #93318e; }
		.ocs-modal-bg { background: rgba(147,49,142,0.08); }
		.ocs-btn { background: #93318e; color: #fff; }
		.ocs-btn:hover { background: #7a2676; }
		.ocs-input { border: 1px solid #93318e; }
		.ocs-checkbox:checked { accent-color: #93318e; }
	</style>
</head>
<body class="min-h-screen flex items-center justify-center">
	<!-- Modal -->
	<div id="modal" class="fixed inset-0 flex items-center justify-center ocs-modal-bg z-10">
		<div class="bg-white rounded-lg shadow-lg p-8 w-full max-w-md border-2 border-[#93318e]">
			<div class="flex flex-col items-center mb-4">
				<div class="rounded-full bg-[#93318e] w-16 h-16 flex items-center justify-center mb-2">
					<span class="text-3xl font-bold text-white">OCS</span>
				</div>
				<h2 class="text-2xl font-bold ocs-purple-text mb-2">Sign-in to OCS</h2>
			</div>
			<form id="loginForm" class="flex flex-col gap-3">
				<input id="username" class="ocs-input rounded px-3 py-2" type="text" placeholder="Username" required autofocus>
				<input id="password" class="ocs-input rounded px-3 py-2" type="password" placeholder="Password" required>
				<button type="submit" class="ocs-btn rounded py-2 font-semibold mt-2">Login</button>
				<div id="loginError" class="text-red-600 text-sm mt-1 hidden"></div>
			</form>
		</div>
	</div>

	<!-- Confirmation Modal -->
	<div id="confirmModal" class="fixed inset-0 flex items-center justify-center ocs-modal-bg z-10 hidden">
		<div class="bg-white rounded-lg shadow-lg p-8 w-full max-w-md border-2 border-[#93318e]">
			<div class="flex flex-col items-center mb-4">
				<div class="rounded-full bg-[#93318e] w-16 h-16 flex items-center justify-center mb-2">
					<span class="text-3xl font-bold text-white">OCS</span>
				</div>
				<h2 class="text-xl font-bold ocs-purple-text mb-2">Delete Computer</h2>
				<div class="text-center text-gray-700 mb-2">
					You are about to delete computer <span class="font-bold" id="compName"></span> from OCS Inventory.<br>
					Please complete validation steps below.
				</div>
			</div>
			<form id="confirmForm" class="flex flex-col gap-3">
				<div class="flex items-center gap-2">
					<span id="captchaQ" class="font-semibold"></span>
					<span>=</span>
					<input id="captchaA" class="ocs-input rounded px-2 py-1 w-20" type="text" required autocomplete="off">
				</div>
				<label class="flex items-center gap-2">
					<input id="confirmCheck" type="checkbox" class="ocs-checkbox" required>
					<span class="text-sm">I Understand and confirm this deletion</span>
				</label>
				<button type="submit" class="ocs-btn rounded py-2 font-semibold mt-2">Delete</button>
				<div id="confirmError" class="text-red-600 text-sm mt-1 hidden"></div>
			</form>
		</div>
	</div>

	<!-- Success Modal -->
	<div id="successModal" class="fixed inset-0 flex items-center justify-center ocs-modal-bg z-10 hidden">
		<div class="bg-white rounded-lg shadow-lg p-8 w-full max-w-md border-2 border-[#93318e] flex flex-col items-center">
			<div class="rounded-full bg-[#93318e] w-16 h-16 flex items-center justify-center mb-2">
				<svg width="32" height="32" fill="none" stroke="#fff" stroke-width="3" viewBox="0 0 24 24"><path d="M5 13l4 4L19 7"/></svg>
			</div>
			<div class="text-xl font-bold ocs-purple-text mb-2">Success</div>
			<div class="text-center text-gray-700 mb-4" id="successMsg"></div>
			<button onclick="location.reload()" class="ocs-btn rounded py-2 px-6 font-semibold">OK</button>
		</div>
	</div>

	<script>
		// Get computer name from query param
		function getQueryParam(name) {
			const url = new URL(window.location.href);
			return url.searchParams.get(name);
		}
		const compName = getQueryParam('name') || '';
		document.getElementById('compName').textContent = compName;

		// Captcha
		let captchaX = Math.floor(Math.random()*10+1), captchaY = Math.floor(Math.random()*10+1);
		document.getElementById('captchaQ').textContent = captchaX + " + " + captchaY;

		// State
		let jwtToken = '';

		// Login form
		document.getElementById('loginForm').onsubmit = async function(e) {
			e.preventDefault();
			const username = document.getElementById('username').value.trim();
			const password = document.getElementById('password').value;
			const errDiv = document.getElementById('loginError');
			errDiv.classList.add('hidden');
			try {
				const res = await fetch('/auth-token', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ username, password })
				});
				const data = await res.json();
				if (!res.ok) throw new Error(data.error || 'Login failed');
				jwtToken = data.token;
				document.getElementById('modal').classList.add('hidden');
				document.getElementById('confirmModal').classList.remove('hidden');
			} catch (err) {
				errDiv.textContent = err.message;
				errDiv.classList.remove('hidden');
			}
		};

		// Confirm form
		document.getElementById('confirmForm').onsubmit = async function(e) {
			e.preventDefault();
			const answer = document.getElementById('captchaA').value.trim();
			const errDiv = document.getElementById('confirmError');
			errDiv.classList.add('hidden');
			if (parseInt(answer) !== captchaX + captchaY) {
				errDiv.textContent = 'Captcha salah!';
				errDiv.classList.remove('hidden');
				return;
			}
			if (!document.getElementById('confirmCheck').checked) {
				errDiv.textContent = 'Anda harus konfirmasi penghapusan.';
				errDiv.classList.remove('hidden');
				return;
			}
			try {
				const res = await fetch("/delete-computer?name=" + encodeURIComponent(compName), {
					method: 'GET',
					headers: { 'Authorization': 'Bearer ' + jwtToken }
				});
				const data = await res.json();
				if (!res.ok) throw new Error(data.error || 'Delete failed');
				document.getElementById('confirmModal').classList.add('hidden');
				document.getElementById('successMsg').textContent = '"' + compName + '" Successfully Removed from OCS Inventory.';
				document.getElementById('successModal').classList.remove('hidden');
			} catch (err) {
				errDiv.textContent = err.message;
				errDiv.classList.remove('hidden');
			}
		};
	</script>
</body>
</html>
`
