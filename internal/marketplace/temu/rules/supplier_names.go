package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"unicode/utf8"
)

func supplierNameLimits() (map[int64]int, Source, error) {
	raw, err := officialData.ReadFile("data/supplier-name-limits.tsv")
	if err != nil {
		return nil, Source{}, ErrOfficialSnapshotInvalid
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 21 || strings.TrimSpace(lines[0]) != "cat1Id\tlimit" {
		return nil, Source{}, ErrOfficialSnapshotInvalid
	}
	values := map[int64]int{}
	for _, line := range lines[1:] {
		fields := strings.Split(strings.TrimSpace(line), "\t")
		if len(fields) != 2 {
			return nil, Source{}, ErrOfficialSnapshotInvalid
		}
		id, e := strconv.ParseInt(fields[0], 10, 64)
		limit, e2 := strconv.Atoi(fields[1])
		if e != nil || e2 != nil || id < 1 || limit < 1 || values[id] != 0 {
			return nil, Source{}, ErrOfficialSnapshotInvalid
		}
		values[id] = limit
	}
	hash := sha256.Sum256(raw)
	return values, Source{EvidenceID: "E45", URL: "https://agentpartner.temu.com/document?cataId=875196199516&docId=899323233508", UpdatedAt: "2026-05-25 16:55:26", ContentHash: hex.EncodeToString(hash[:]), TotalRows: 20, IncludedRows: 20}, nil
}

func SupplierNameSource() (Source, error) { _, source, err := supplierNameLimits(); return source, err }

// E45 counts ASCII letters/digits/English punctuation as one and other
// characters as two. Its unlisted first-level categories explicitly use 500.
func SupplierNameAllowed(name string, cat1ID int64) bool {
	if !utf8.ValidString(name) || strings.TrimSpace(name) == "" || strings.ContainsRune(name, 0) || cat1ID < 1 {
		return false
	}
	limits, _, err := supplierNameLimits()
	if err != nil {
		return false
	}
	maximum := limits[cat1ID]
	if maximum == 0 {
		maximum = 500
	}
	length := 0
	for _, r := range name {
		if r >= 33 && r <= 126 {
			length++
		} else {
			length += 2
		}
		if length > maximum {
			return false
		}
	}
	return true
}
