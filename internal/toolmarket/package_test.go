package toolmarket

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const packageTestApp = "https://capture.example.com:8443/capture/1688"

func currentPackageFiles(t *testing.T) map[string]string {
	t.Helper()
	u, err := url.Parse(packageTestApp)
	require.NoError(t, err)
	manifest, err := json.Marshal(map[string]any{
		"manifest_version": 3, "name": "1688 / Amazon 商品采集", "version": "0.2.0",
		"externally_connectable": map[string]any{"matches": []string{
			"https://" + u.Hostname() + "/capture/1688", "https://" + u.Hostname() + "/capture/amazon",
		}},
	})
	require.NoError(t, err)
	return map[string]string{
		"manifest.json": string(manifest),
		"background.js": `const app="` + packageTestApp + `";const amazon="https://capture.example.com:8443/capture/amazon";`,
		"popup.js":      "x", "extractor.js": "x", "page.js": "x", "popup.html": "x", "popup.css": "x",
	}
}

func writePackage(t *testing.T, files map[string]string) PackageConfig {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for n, v := range files {
		f, e := w.Create(n)
		require.NoError(t, e)
		_, e = f.Write([]byte(v))
		require.NoError(t, e)
	}
	require.NoError(t, w.Close())
	return writePackageArchive(t, b.Bytes())
}

func writePackageArchive(t *testing.T, raw []byte) PackageConfig {
	t.Helper()
	p := filepath.Join(t.TempDir(), "plugin.zip")
	require.NoError(t, os.WriteFile(p, raw, 0600))
	sum := sha256.Sum256(raw)
	return PackageConfig{Path: p, SHA256: hex.EncodeToString(sum[:]), CaptureAppURL: packageTestApp}
}

func TestPackageRequiresTrustedFullAddressAndDigest(t *testing.T) {
	cfg := writePackage(t, currentPackageFiles(t))
	packageData, e := LoadPackage(cfg)
	require.NoError(t, e)
	raw, e := os.ReadFile(cfg.Path)
	require.NoError(t, e)
	require.Equal(t, raw, packageData)
	cfg.CaptureAppURL = "https://capture.example.com:9443/capture/1688"
	_, e = LoadPackage(cfg)
	require.ErrorIs(t, e, ErrUnavailable)
	cfg.CaptureAppURL = packageTestApp
	cfg.SHA256 = "bad"
	_, e = LoadPackage(cfg)
	require.ErrorIs(t, e, ErrUnavailable)
}

func TestPackageRejectsLayoutsAndReceiverBindingsOutsideCurrentBuild(t *testing.T) {
	for name, mutate := range map[string]func(map[string]string){
		"missing page script":         func(f map[string]string) { delete(f, "page.js") },
		"extra file":                  func(f map[string]string) { f["other.js"] = "x" },
		"path instead of page script": func(f map[string]string) { delete(f, "page.js"); f["../page.js"] = "x" },
		"fixture name": func(f map[string]string) {
			f["manifest.json"] = strings.ReplaceAll(f["manifest.json"], "1688 / Amazon 商品采集", "1688 / Amazon 商品采集 — 任务 Fixture")
		},
		"wrong manifest version": func(f map[string]string) {
			f["manifest.json"] = strings.ReplaceAll(f["manifest.json"], `"manifest_version":3`, `"manifest_version":2`)
		},
		"invalid manifest": func(f map[string]string) { f["manifest.json"] = "{" },
		"duplicate receiver": func(f map[string]string) {
			f["manifest.json"] = strings.ReplaceAll(f["manifest.json"], "/capture/amazon", "/capture/1688")
		},
		"other receiver host": func(f map[string]string) {
			f["manifest.json"] = strings.ReplaceAll(f["manifest.json"], "https://capture.example.com/capture/amazon", "https://other.example.com/capture/amazon")
		},
		"receiver wildcard": func(f map[string]string) {
			f["manifest.json"] = strings.ReplaceAll(f["manifest.json"], "/capture/amazon", "/capture/*")
		},
		"missing compiled Amazon receiver": func(f map[string]string) { f["background.js"] = `const app="` + packageTestApp + `";` },
		"other compiled Amazon port": func(f map[string]string) {
			f["background.js"] = strings.ReplaceAll(f["background.js"], ":8443/capture/amazon", ":9443/capture/amazon")
		},
		"other compiled 1688 port": func(f map[string]string) {
			f["background.js"] = strings.ReplaceAll(f["background.js"], ":8443/capture/1688", ":9443/capture/1688")
		},
		"oversized member": func(f map[string]string) { f["page.js"] = strings.Repeat("x", MaxPackageBytes+1) },
		"oversized total": func(f map[string]string) {
			for _, n := range []string{"page.js", "popup.js", "extractor.js", "popup.html", "popup.css"} {
				f[n] = strings.Repeat("x", MaxPackageBytes)
			}
		},
		"retired single-platform layout": func(f map[string]string) {
			delete(f, "page.js")
			f["manifest.json"] = `{"manifest_version":3,"name":"1688 商品采集","externally_connectable":{"matches":["https://capture.example.com/capture/1688"]}}`
		},
	} {
		t.Run(name, func(t *testing.T) {
			files := currentPackageFiles(t)
			mutate(files)
			_, err := LoadPackage(writePackage(t, files))
			require.ErrorIs(t, err, ErrUnavailable)
		})
	}
}

func TestPackageRejectsDuplicateOrSymlinkMember(t *testing.T) {
	for _, name := range []string{"duplicate", "symlink"} {
		t.Run(name, func(t *testing.T) {
			var archive bytes.Buffer
			writer := zip.NewWriter(&archive)
			for filename, contents := range currentPackageFiles(t) {
				header := &zip.FileHeader{Name: filename, Method: zip.Deflate}
				if filename == "page.js" {
					if name == "duplicate" {
						header.Name = "popup.js"
					} else {
						header.SetMode(os.ModeSymlink | 0600)
					}
				}
				entry, err := writer.CreateHeader(header)
				require.NoError(t, err)
				_, err = entry.Write([]byte(contents))
				require.NoError(t, err)
			}
			require.NoError(t, writer.Close())
			_, err := LoadPackage(writePackageArchive(t, archive.Bytes()))
			require.ErrorIs(t, err, ErrUnavailable)
		})
	}
}

func TestDeploymentBuildPackage(t *testing.T) {
	directory := os.Getenv("TOOLMARKET_PACKAGE_TEST_DIR")
	if directory == "" {
		t.Skip("requires actual existing extension release build")
	}
	raw, e := os.ReadFile(filepath.Join(directory, "release.json"))
	require.NoError(t, e)
	var record struct {
		CaptureAppURL string `json:"captureAppUrl"`
		SHA256        string `json:"sha256"`
		Filename      string `json:"filename"`
	}
	require.NoError(t, json.Unmarshal(raw, &record))
	cfg := PackageConfig{Path: filepath.Join(directory, record.Filename), CaptureAppURL: record.CaptureAppURL, SHA256: record.SHA256}
	bytes, e := LoadPackage(cfg)
	require.NoError(t, e)
	require.NotEmpty(t, bytes)
	cfg.CaptureAppURL = "https://other.shuomi-test.com/capture/1688"
	_, e = LoadPackage(cfg)
	require.ErrorIs(t, e, ErrUnavailable)
}
