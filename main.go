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
	log.Printf("[SUCCESS] AD Manager Plus - Data successfully parsed, Total: %d", len(cleanData))

	// 7. Cek koneksi OCS MySQL dan tampilkan list komputer
	ocsCfg := client.LoadOCSConfig()
	ocsClient, err := client.NewOCSMySQLClient(ocsCfg)
	if err != nil {
		fmt.Println("[ERROR] OCS - Koneksi gagal:", err)
		return
	}
	if err := ocsClient.Ping(); err != nil {
		fmt.Println("[ERROR] OCS - Autentikasi gagal:", err)
		return
	}
	log.Println("[SUCCESS] OCS - Autentication Success!")

	// List komputer OCS
	ocsComputers, err := parser.ListOCSComputers(ocsClient.DB, 0)
	if err != nil {
		log.Printf("[ERROR] Gagal mengambil data komputer OCS: %v", err)
		return
	}
	log.Printf("[SUCCESS] OCS - Data successfully parsed, Total: %d", len(ocsComputers))

	// Gabungkan data OCS dan AD
	finalList := parser.CombineOCSAndAD(ocsComputers, cleanData)
	log.Printf("[SUCCESS] OCS x AD - Combined Data, Total: %d", len(finalList))

	// Simpan ke Elasticsearch
	esCfg := client.LoadElasticsearchConfig()
	esClient, err := client.NewElasticsearchClient(esCfg)
	if err != nil {
		log.Printf("[ERROR] Gagal membuat client Elasticsearch: %v", err)
		return
	}

	// --- Sinkronisasi dua arah: hapus data yang sudah tidak ada di OCS/AD ---
	// 1. Build cache OCS dan AD (hashing)
	cacheOCS := make(map[string]struct{})
	cacheAD := make(map[string]struct{})
	hashName := func(name string) string {
		return parser.HashComputerName(name)
	}
	for _, ocs := range ocsComputers {
		cacheOCS[hashName(ocs.ComputerName)] = struct{}{}
	}
	for _, ad := range cleanData {
		cacheAD[hashName(ad.ComputerName)] = struct{}{}
	}

	// 2. Ambil semua document ID dari Elasticsearch
	// (gunakan search API, ambil hanya field _id)
	var esIDs []string
	{
		// Ambil 10.000 data per request (bisa dioptimasi scroll/bulk jika data sangat besar)
		from := 0
		size := 10000
		for {
			query := `{"query":{"match_all":{}},"_source":false,"from":` + fmt.Sprintf("%d", from) + `,"size":` + fmt.Sprintf("%d", size) + `}`
			res, err := esClient.Client.Search(
				esClient.Client.Search.WithIndex(esCfg.Index),
				esClient.Client.Search.WithBody(strings.NewReader(query)),
			)
			if err != nil {
				log.Printf("[ERROR] Gagal mengambil document ID dari Elasticsearch: %v", err)
				break
			}
			defer res.Body.Close()
			var resp struct {
				Hits struct {
					Hits []struct {
						ID string `json:"_id"`
					} `json:"hits"`
				} `json:"hits"`
			}
			if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
				log.Printf("[ERROR] Gagal decode response Elasticsearch: %v", err)
				break
			}
			if len(resp.Hits.Hits) == 0 {
				break
			}
			for _, hit := range resp.Hits.Hits {
				esIDs = append(esIDs, hit.ID)
			}
			if len(resp.Hits.Hits) < size {
				break
			}
			from += size
		}
	}

	// 3. Hapus document yang tidak ada di OCS dan tidak ada di AD
	deleted := 0
	for _, id := range esIDs {
		h := hashName(id)
		_, existsOCS := cacheOCS[h]
		_, existsAD := cacheAD[h]
		if !existsOCS && !existsAD {
			res, err := esClient.Client.Delete(esCfg.Index, id)
			if err != nil {
				log.Printf("[ERROR] Gagal hapus document %s di Elasticsearch: %v", id, err)
				continue
			}
			if res.IsError() {
				log.Printf("[ERROR] Elasticsearch response error saat hapus %s: %s", id, res.String())
			} else {
				deleted++
			}
			res.Body.Close()
		}
	}
	log.Printf("[INFO] OCS x AD - Remove Deleted Data, Total: %d", deleted)

	// --- Index/update data yang masih ada ---
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
	log.Printf("[INFO] Indexing Finished. Success: %d, Failed: %d", success, failed)
}
