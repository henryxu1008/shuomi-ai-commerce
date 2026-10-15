package currentapplication

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/knowledge"
)

const minioCapacityTestID = "d7b806bb-6f4f-47be-999c-99f027252244"

func TestLoadConfigAcceptsExplicitMinIOKnowledgeCapacityIdentity(t *testing.T) {
	cfg := knowledgeRuntimeTestConfig()
	cfg.Knowledge.Storage.Endpoint = "http://knowledge-objects:9000"
	b, err := json.Marshal(cfg)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(b, &body))
	storage := body["knowledge"].(map[string]any)["storage"].(map[string]any)
	storage["capacityIdentityProvider"] = "minio-deployment"
	storage["capacityMinIODeploymentID"] = minioCapacityTestID
	b, err = json.Marshal(body)
	require.NoError(t, err)
	_, err = LoadConfig(writeManifest(t, string(b)))
	require.NoError(t, err, "explicit MinIO identity must be consumed by the original private manifest path")
}

func TestMinIOCapacityConfigRejectsIncompleteOrForeignIdentity(t *testing.T) {
	for _, kind := range []string{"knowledge", "ecoservices"} {
		for _, scenario := range []string{"unknown_provider", "missing_pin", "pin_without_provider", "nil_uuid", "foreign_endpoint", "cos"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				storage := KnowledgeStorageConfig{Endpoint: "http://knowledge-objects:9000", Region: "us-east-1", Bucket: kind, Mode: "aws", AccessKeyID: "runtime", SecretAccessKey: "synthetic", CapacityIdentityProvider: "minio-deployment", CapacityMinIODeploymentID: minioCapacityTestID}
				switch scenario {
				case "unknown_provider":
					storage.CapacityIdentityProvider = "unknown"
				case "missing_pin":
					storage.CapacityMinIODeploymentID = ""
				case "pin_without_provider":
					storage.CapacityIdentityProvider = ""
				case "nil_uuid":
					storage.CapacityMinIODeploymentID = "00000000-0000-0000-0000-000000000000"
				case "foreign_endpoint":
					storage.Endpoint = "http://10.0.0.2:9000"
				case "cos":
					storage.Mode = "cos"
					storage.COSImmutableNonVersionedBucketPolicy = true
				}
				if kind == "knowledge" {
					c := knowledgeRuntimeTestConfig()
					c.Knowledge.Storage = storage
					require.Error(t, c.validateKnowledge())
				} else {
					c := runtimeTestConfig()
					db := c.SourceAccountDatabase
					db.Database = "ecoservices"
					db.User = "ecoservices_runtime"
					db.MaxConnections = 4
					c.Ecoservices = &EcoservicesConfig{Enabled: true, NonPaymentOnly: true, Database: db, Storage: storage}
					require.Error(t, c.validateEcoservices())
				}
			})
		}
	}
}

func TestStorageMinIOConsumersKeepNamespaceOnRestartAndRefuseChangedDeployment(t *testing.T) {
	for _, kind := range []string{"knowledge", "ecoservices"} {
		t.Run(kind, func(t *testing.T) {
			var changed atomic.Bool
			var listCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/minio/admin/v3/info" {
					w.Header().Set("Content-Type", "application/json")
					id := minioCapacityTestID
					if changed.Load() {
						id = "91cac44d-7030-4f4a-a76a-4474af1e73bb"
					}
					_, _ = w.Write([]byte(`{"deploymentID":"` + id + `"}`))
					return
				}
				require.False(t, r.URL.Query().Has("acl"), "explicit MinIO must not call dummy bucket ACL")
				w.Header().Set("Content-Type", "application/xml")
				if r.URL.Query().Has("versioning") {
					_, _ = w.Write([]byte(`<VersioningConfiguration/>`))
					return
				}
				listCalls.Add(1)
				_, _ = w.Write([]byte(`<ListBucketResult><IsTruncated>false</IsTruncated><KeyCount>0</KeyCount></ListBucketResult>`))
			}))
			defer server.Close()
			c, db := storageStartupFixture(t)
			storage := KnowledgeStorageConfig{Endpoint: server.URL, Region: "us-east-1", Bucket: kind, Mode: "aws", AccessKeyID: "runtime", SecretAccessKey: "synthetic", CapacityIdentityProvider: "minio-deployment", CapacityMinIODeploymentID: minioCapacityTestID}
			if kind == "knowledge" {
				c.Knowledge = &KnowledgeConfig{Enabled: true, Storage: storage}
			} else {
				c.Ecoservices = &EcoservicesConfig{Enabled: true, NonPaymentOnly: true, Storage: storage}
			}
			start := func() error {
				return prepareStorageCapacity(context.Background(), c, db, &knowledge.Service{}, &knowledge.Processor{}, nil, &EcoservicesRuntime{})
			}
			require.NoError(t, start())
			require.Equal(t, int32(1), listCalls.Load())
			require.NoError(t, start())
			require.Equal(t, int32(1), listCalls.Load(), "retained namespace must not be rescanned")
			changed.Store(true)
			require.Error(t, start())
			require.Equal(t, int32(1), listCalls.Load())
			var rows []struct{ Prefix string }
			require.NoError(t, db.Table("storage_capacity_namespaces").Select("prefix").Scan(&rows).Error)
			require.Len(t, rows, 1)
			prefix := kind + "/"
			if strings.EqualFold(kind, "ecoservices") {
				prefix = "ecoservices/files/"
			}
			require.Equal(t, prefix, rows[0].Prefix)
		})
	}
}
