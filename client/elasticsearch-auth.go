// client/elasticsearch-auth.go
package client

import (
	"fmt"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
)

// ElasticsearchConfig menyimpan konfigurasi koneksi Elasticsearch
type ElasticsearchConfig struct {
	URL      string
	Username string
	Password string
	Index    string
}

func LoadElasticsearchConfig() ElasticsearchConfig {
	return ElasticsearchConfig{
		URL:      os.Getenv("ELASTICSEARCH_SERVER_URL"),
		Username: os.Getenv("ELASTICSEARCH_USER"),
		Password: os.Getenv("ELASTICSEARCH_PASS"),
		Index:    os.Getenv("ELASTICSEARCH_INDEX_NAME"),
	}
}

// ElasticsearchClient adalah client untuk koneksi Elasticsearch
type ElasticsearchClient struct {
	Client *elasticsearch.Client
	Config ElasticsearchConfig
}

func NewElasticsearchClient(cfg ElasticsearchConfig) (*ElasticsearchClient, error) {
	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.URL},
		Username:  cfg.Username,
		Password:  cfg.Password,
	}
	es, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat client elasticsearch: %v", err)
	}
	return &ElasticsearchClient{Client: es, Config: cfg}, nil
}
