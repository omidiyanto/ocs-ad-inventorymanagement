// main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"ocs-ad-inventorymanagement/api"
	"ocs-ad-inventorymanagement/client"
	"ocs-ad-inventorymanagement/parser"
	"ocs-ad-inventorymanagement/web"

	"github.com/gin-gonic/gin"
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

	// Jalankan web API Gin untuk delete-computer secara async
	ocsCfg := client.LoadOCSConfig()
	ocsClient, err := client.NewOCSMySQLClient(ocsCfg)
	if err != nil {
		log.Fatalf("[ERROR] OCS - Koneksi gagal: %v", err)
	}
	if err := ocsClient.Ping(); err != nil {
		log.Fatalf("[ERROR] OCS - Autentikasi gagal: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	basePath := os.Getenv("BASE_PATH_URL")
	if basePath == "" {
		basePath = "/ocsextra"
	}
	// Pastikan basePath diawali dengan /
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	// Pastikan basePath tidak diakhiri / (kecuali root)
	if len(basePath) > 1 && strings.HasSuffix(basePath, "/") {
		basePath = strings.TrimRight(basePath, "/")
	}

	apiGroup := r.Group(basePath + "/api")
	apiGroup.POST("/auth-token", api.AuthTokenHandler)
	apiGroup.POST("/delete-computer", api.DeleteComputerHandler(ocsClient.DB))

	// Frontend GET
	r.GET(basePath+"/delete-computer", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, web.FrontendHTML)
	})

	// Ambil port dari env, default 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	go func() {
		if err := r.Run(addr); err != nil {
			log.Fatalf("[ERROR] Gagal menjalankan web API: %v", err)
		}
	}()

	// Scheduler: jalankan sinkronisasi setiap 90 detik
	for {
		// 1. List komputer OCS
		ocsComputers, err := parser.ListOCSComputers(ocsClient.DB, 0)
		if err != nil {
			log.Printf("[ERROR] Gagal mengambil data komputer OCS: %v", err)
			continue
		}
		log.Printf("[SUCCESS] OCS - Data successfully parsed, Total: %d", len(ocsComputers))

		// 2. Gabungkan data OCS dan AD
		finalList := parser.CombineOCSAndAD(ocsComputers, cleanData)
		log.Printf("[SUCCESS] OCS x AD - Combined Data, Total: %d", len(finalList))

		// 3. Simpan ke Elasticsearch
		esCfg := client.LoadElasticsearchConfig()
		esClient, err := client.NewElasticsearchClient(esCfg)
		if err != nil {
			log.Printf("[ERROR] Gagal membuat client Elasticsearch: %v", err)
			continue
		}

		// --- Sinkronisasi dua arah: hapus data yang sudah tidak ada di OCS/AD ---
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

		var esIDs []string
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

		// --- Index/update data yang masih ada, gunakan batch dan paralel (goroutine) ---
		batchSize := 500
		maxParallel := 4
		type indexResult struct {
			success int
			failed  int
		}
		indexBatch := func(batch []parser.FinalComputerRow, ch chan<- indexResult) {
			success, failed := 0, 0
			for _, row := range batch {
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
			ch <- indexResult{success, failed}
		}

		var batches [][]parser.FinalComputerRow
		for i := 0; i < len(finalList); i += batchSize {
			end := i + batchSize
			if end > len(finalList) {
				end = len(finalList)
			}
			batches = append(batches, finalList[i:end])
		}

		ch := make(chan indexResult, len(batches))
		sem := make(chan struct{}, maxParallel)

		for _, batch := range batches {
			sem <- struct{}{} // acquire
			go func(b []parser.FinalComputerRow) {
				defer func() { <-sem }() // release
				indexBatch(b, ch)
			}(batch)
		}

		success, failed := 0, 0
		for i := 0; i < len(batches); i++ {
			res := <-ch
			success += res.success
			failed += res.failed
		}
		log.Printf("[INFO] Indexing Finished (Batch Parallel). Success: %d, Failed: %d", success, failed)

		time.Sleep(90 * time.Second)
	}
}
