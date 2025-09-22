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

func main() {
	// Memuat file .env, tidak akan error jika file tidak ada
	godotenv.Load()

	// 1. Muat konfigurasi LDAP dan konek
	ldapCfg := client.LoadLDAPConfig()
	ldapClient, err := client.NewLDAPClient(ldapCfg)
	if err != nil {
		log.Fatalf("[FATAL] Gagal koneksi ke LDAP: %v", err)
	}
	defer ldapClient.Close()
	log.Println("[SUCCESS] LDAP - Berhasil konek dan autentikasi ke Active Directory.")

	// 2. Koneksi ke OCS MySQL (tetap sama)
	ocsCfg := client.LoadOCSConfig()
	ocsClient, err := client.NewOCSMySQLClient(ocsCfg)
	if err != nil {
		log.Fatalf("[FATAL] OCS - Koneksi gagal: %v", err)
	}
	if err := ocsClient.Ping(); err != nil {
		log.Fatalf("[FATAL] OCS - Ping database gagal: %v", err)
	}
	log.Println("[SUCCESS] OCS - Berhasil konek ke database.")

	// 3. Jalankan Web API (tetap sama)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	basePath := os.Getenv("BASE_PATH_URL")
	if basePath == "" {
		basePath = "/ocsextra"
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	if len(basePath) > 1 && strings.HasSuffix(basePath, "/") {
		basePath = strings.TrimRight(basePath, "/")
	}

	apiGroup := r.Group(basePath + "/api")
	apiGroup.POST("/auth-token", api.AuthTokenHandler)
	apiGroup.POST("/delete-computer", api.DeleteComputerHandler(ocsClient.DB))

	r.GET(basePath+"/delete-computer", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, web.FrontendHTML)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	go func() {
		log.Printf("[INFO] Web server berjalan di alamat %s%s", addr, basePath)
		if err := r.Run(addr); err != nil {
			log.Fatalf("[FATAL] Gagal menjalankan web API: %v", err)
		}
	}()

	// 4. Scheduler utama untuk sinkronisasi data
	for {
		// --- Ambil data dari AD via LDAP ---
		ldapEntries, err := ldapClient.ListComputers()
		if err != nil {
			// Jika error, coba re-koneksi sekali sebelum gagal
			log.Printf("[ERROR] Gagal mengambil data dari LDAP: %v. Mencoba re-koneksi...", err)
			ldapClient.Close()
			ldapClient, err = client.NewLDAPClient(ldapCfg)
			if err != nil {
				log.Fatalf("[FATAL] Gagal re-koneksi ke LDAP: %v", err)
			}
			ldapEntries, err = ldapClient.ListComputers()
			if err != nil {
				log.Fatalf("[FATAL] Gagal mengambil data dari LDAP setelah re-koneksi: %v", err)
			}
		}

		// --- Parse data AD ---
		cleanData, err := parser.ParseComputerReportFromLDAP(ldapEntries)
		if err != nil {
			log.Fatalf("Proses transformasi data AD gagal: %v", err)
		}
		log.Printf("[SUCCESS] LDAP - Data berhasil diparsing, Total: %d", len(cleanData))

		// --- Ambil data dari OCS ---
		ocsComputers, err := parser.ListOCSComputers(ocsClient.DB, 0)
		if err != nil {
			log.Printf("[ERROR] Gagal mengambil data komputer OCS: %v", err)
			continue
		}
		log.Printf("[SUCCESS] OCS - Data berhasil diparsing, Total: %d", len(ocsComputers))

		// --- Gabungkan data OCS dan AD ---
		finalList := parser.CombineOCSAndAD(ocsComputers, cleanData)
		log.Printf("[SUCCESS] OCS x AD - Data digabungkan, Total: %d", len(finalList))

		// --- Simpan ke Elasticsearch ---
		esCfg := client.LoadElasticsearchConfig()
		esClient, err := client.NewElasticsearchClient(esCfg)
		if err != nil {
			log.Printf("[ERROR] Gagal membuat client Elasticsearch: %v", err)
			continue
		}

		// --- Sinkronisasi: hapus data yang sudah tidak ada di OCS/AD ---
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
		log.Printf("[INFO] Elasticsearch - Hapus data lama, Total: %d", deleted)

		// --- Index/update data yang masih ada ---
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
		log.Printf("[INFO] Elasticsearch - Indexing selesai. Sukses: %d, Gagal: %d", success, failed)

		log.Println("----------------- Siklus Selesai, Menunggu 60 Detik -----------------")
		time.Sleep(60 * time.Second)
	}
}
