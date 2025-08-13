// main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"ocs-ad-inventorymanagement/client" // Ganti 'ocs-ad-inventorymanagement' sesuai nama modul Anda
	"ocs-ad-inventorymanagement/parser" // Ganti 'ocs-ad-inventorymanagement' sesuai nama modul Anda
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// loadConfig memuat konfigurasi dari variabel lingkungan.
func loadConfig() (client.Config, error) {
	// Memuat file .env, tidak akan error jika file tidak ada
	godotenv.Load()

	cfg := client.Config{}
	requiredVars := map[string]*string{
		"AD_BASE_URL":           &cfg.BaseURL,
		"AD_USERNAME":           &cfg.Username,
		"AD_ENCRYPTED_PASSWORD": &cfg.EncryptedPassword,
		"AD_REPORT_ID":          &cfg.ReportID,
		"AD_GENERATION_ID":      &cfg.GenerationID,
	}

	for key, valuePtr := range requiredVars {
		value := os.Getenv(key)
		if value == "" {
			return cfg, fmt.Errorf("variabel lingkungan wajib '%s' tidak diatur", key)
		}
		*valuePtr = value
	}
	return cfg, nil
}

func main() {
	// 1. Muat konfigurasi dari .env
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Gagal memuat konfigurasi: %v", err)
	}

	// 2. Buat client baru dengan konfigurasi yang sudah dimuat
	adClient, err := client.New(cfg)
	if err != nil {
		log.Fatalf("Gagal membuat client: %v", err)
	}

	// 3. Jalankan login menggunakan method dari client
	if err := adClient.Login(); err != nil {
		log.Fatalf("Proses login gagal: %v", err)
	}

	// 4. Ambil laporan
	rawData, err := adClient.FetchComputerReport()
	if err != nil {
		log.Fatalf("Proses pengambilan laporan gagal: %v", err)
	}

	// 5. Ubah data
	cleanData, err := parser.ParseComputerReport(rawData)
	if err != nil {
		log.Fatalf("Proses transformasi data gagal: %v", err)
	}

	// 6. Log hasil akhir
	log.Printf("[INFO] Data AD berhasil diparsing, jumlah: %d", len(cleanData))

	// 7. Cek koneksi OCS MySQL dan tampilkan list komputer
	ocsCfg := client.LoadOCSConfig()
	ocsClient, err := client.NewOCSMySQLClient(ocsCfg)
	if err != nil {
		fmt.Println("[OCS MySQL] Koneksi gagal:", err)
		return
	}
	if err := ocsClient.Ping(); err != nil {
		fmt.Println("[OCS MySQL] Autentikasi gagal:", err)
		return
	}
	fmt.Println("[OCS MySQL] Autentikasi berhasil!")

	// List komputer OCS
	ocsComputers, err := parser.ListOCSComputers(ocsClient.DB, 0)
	if err != nil {
		log.Printf("[ERROR] Gagal mengambil data komputer OCS: %v", err)
		return
	}
	log.Printf("[INFO] Data OCS berhasil diambil, jumlah: %d", len(ocsComputers))

	// Gabungkan data OCS dan AD
	finalList := parser.CombineOCSAndAD(ocsComputers, cleanData)
	log.Printf("[INFO] Data gabungan OCS+AD siap, jumlah: %d", len(finalList))

	// Simpan ke Elasticsearch
	esCfg := client.LoadElasticsearchConfig()
	esClient, err := client.NewElasticsearchClient(esCfg)
	if err != nil {
		log.Printf("[ERROR] Gagal membuat client Elasticsearch: %v", err)
		return
	}

	// Indexing per row (bulk bisa dioptimasi nanti)
	success, failed := 0, 0
	for _, row := range finalList {
		docID := row.ComputerName
		body, _ := json.Marshal(row)
		res, err := esClient.Client.Index(esCfg.Index, strings.NewReader(string(body)), esClient.Client.Index.WithDocumentID(docID))
		if err != nil {
			log.Printf("[ERROR] Indexing gagal untuk %s: %v", docID, err)
			failed++
			continue
		}
		if res.IsError() {
			log.Printf("[ERROR] Elasticsearch response error untuk %s: %s", docID, res.String())
			failed++
		} else {
			success++
		}
		res.Body.Close()
	}
	log.Printf("[INFO] Indexing selesai. Sukses: %d, Gagal: %d", success, failed)
}
