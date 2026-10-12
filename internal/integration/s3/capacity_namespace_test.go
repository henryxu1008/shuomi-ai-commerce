package s3

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
)

func TestEmptyLiveListingCannotHideHistoricalObjectVersions(t *testing.T) {
	for _, historical := range []bool{true, false} {
		t.Run(map[bool]string{true: "delete_marker", false: "new_empty"}[historical], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				q := r.URL.Query()
				switch {
				case q.Has("versioning"):
					_, _ = w.Write([]byte(`<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>Enabled</Status></VersioningConfiguration>`))
				case q.Has("versions"):
					require.Equal(t, "1", q.Get("max-keys"))
					if historical {
						_, _ = w.Write([]byte(`<ListVersionsResult><IsTruncated>false</IsTruncated><DeleteMarker><Key>knowledge/old</Key><VersionId>old</VersionId><IsLatest>true</IsLatest></DeleteMarker></ListVersionsResult>`))
					} else {
						_, _ = w.Write([]byte(`<ListVersionsResult><IsTruncated>false</IsTruncated></ListVersionsResult>`))
					}
				default:
					require.Equal(t, "1", q.Get("max-keys"))
					_, _ = w.Write([]byte(`<ListBucketResult><IsTruncated>false</IsTruncated><KeyCount>0</KeyCount></ListBucketResult>`))
				}
			}))
			defer server.Close()
			e := VerifyEmptyCapacityPrefix(context.Background(), ClientConfig{Endpoint: server.URL, Region: "us-east-1", AccessKeyID: "synthetic", SecretAccessKey: "synthetic", UsePathStyle: true}, "bucket", "knowledge/", "aws")
			if historical {
				require.ErrorIs(t, e, storagecapacity.ErrConflict)
			} else {
				require.NoError(t, e)
			}
		})
	}
}
