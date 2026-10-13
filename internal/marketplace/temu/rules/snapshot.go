package rules

import (
	"strconv"
	"strings"
)

func snapshotRows(path string, fields, count int) ([][]string, string, error) {
	raw, err := officialData.ReadFile("data/" + path)
	if err != nil {
		return nil, "", ErrOfficialSnapshotInvalid
	}
	return decodeSnapshotRows(raw, fields, count)
}

func decodeSnapshotRows(raw []byte, fields, count int) ([][]string, string, error) {
	// Git checkouts may change line endings on Windows. Rule identity must
	// remain tied to the same official data, not the checkout's operating system.
	canonical := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(strings.TrimSuffix(canonical, "\n"), "\n")
	if len(lines) != count+1 {
		return nil, "", ErrOfficialSnapshotInvalid
	}
	rows := make([][]string, 0, count)
	for _, line := range lines[1:] {
		cells := strings.Split(line, "\t")
		if len(cells) != fields {
			return nil, "", ErrOfficialSnapshotInvalid
		}
		rows = append(rows, cells)
	}
	return rows, contentHash([]byte(canonical)), nil
}

func positiveID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 1 || strconv.FormatInt(id, 10) != value {
		return 0, ErrOfficialSnapshotInvalid
	}
	return id, nil
}
func opening(value string) (Opening, error) {
	switch value {
	case "true":
		return Open, nil
	case "false":
		return Closed, nil
	case "coming soon":
		return ComingSoon, nil
	case "空白":
		return Unknown, nil
	default:
		return "", ErrOfficialSnapshotInvalid
	}
}

func (c *Catalog) readSemi() error {
	rows, hash, err := snapshotRows("semi-sites.tsv", 4, 96)
	if err != nil {
		return err
	}
	for _, row := range rows {
		id, err := positiveID(row[0])
		if err != nil {
			return err
		}
		state, err := opening(row[2])
		if err != nil || state != Open && state != Closed {
			return ErrOfficialSnapshotInvalid
		}
		if _, duplicate := c.semi[id]; duplicate {
			return ErrOfficialSnapshotInvalid
		}
		site := SemiSite{SiteID: id, Label: row[1], Opening: state, Association: row[3], AssociatedSiteIDs: []int64{}}
		switch row[3] {
		case "-", "泛欧", "原文空白":
		case "103,104":
			site.AssociatedSiteIDs = []int64{103, 104}
		default:
			return ErrOfficialSnapshotInvalid
		}
		c.semi[id] = site
	}
	c.sources = append(c.sources, Source{EvidenceID: "E23", URL: "https://agentpartner.temu.com/document?cataId=875196199516&docId=893022798763", UpdatedAt: "2025-09-29", ContentHash: hash, TotalRows: 96, IncludedRows: 96})
	return nil
}

func (c *Catalog) readRegions() error {
	rows, hash, err := snapshotRows("mapped-regions.tsv", 6, 87)
	if err != nil {
		return err
	}
	for _, row := range rows {
		region, err := parseRegion(row)
		if err != nil {
			return err
		}
		if _, duplicate := c.regions[region.RegionID]; duplicate {
			return ErrOfficialSnapshotInvalid
		}
		c.regions[region.RegionID] = region
	}
	c.sources = append(c.sources, Source{EvidenceID: "E30", URL: "https://partner.temu.com/documentation?menu_code=38e79b35d2cb463d85619c1c786dd303&sub_menu_code=07938e60f95344a0a9444f524ea61e31", UpdatedAt: "2026-07-16 00:11:23", ContentHash: hash, TotalRows: 235, IncludedRows: 87})
	return nil
}

func parseRegion(row []string) (Region, error) {
	r := Region{Label: row[2], DeploymentPartition: row[3]}
	var err error
	if r.RegionID, err = positiveID(row[0]); err != nil {
		return Region{}, err
	}
	if r.SiteID, err = positiveID(row[1]); err != nil {
		return Region{}, err
	}
	if r.Semi, err = opening(row[4]); err != nil {
		return Region{}, err
	}
	if r.Local, err = opening(row[5]); err != nil {
		return Region{}, err
	}
	if row[3] != "US" && row[3] != "EU" && row[3] != "GLOBAL" {
		return Region{}, ErrOfficialSnapshotInvalid
	}
	return r, nil
}

func (c *Catalog) readLocalSpecifications() error {
	rows, hash, err := snapshotRows("local-specifications.tsv", 15, 39)
	if err != nil {
		return err
	}
	for _, row := range rows {
		spec, err := parseLocalSpecification(row)
		if err != nil {
			return err
		}
		if _, duplicate := c.local[spec.SiteID]; duplicate {
			return ErrOfficialSnapshotInvalid
		}
		c.local[spec.SiteID] = spec
	}
	c.sources = append(c.sources, Source{EvidenceID: "E24", URL: "https://partner.temu.com/documentation?menu_code=38e79b35d2cb463d85619c1c786dd303&sub_menu_code=2cc05ca389ca4eeb8e704e903ebcc6bd", UpdatedAt: "2026-01-26 15:47:43", ContentHash: hash, TotalRows: 39, IncludedRows: 39})
	return nil
}

func parseLocalSpecification(row []string) (LocalSpecification, error) {
	s := LocalSpecification{Label: row[0], DocumentPartition: row[1], RawLanguages: row[2], BaseCurrency: row[10], RetailCurrency: row[11]}
	var err error
	if s.SiteID, err = positiveID(row[3]); err != nil {
		return LocalSpecification{}, err
	}
	for _, language := range strings.Split(row[2], ",") {
		language = strings.Trim(strings.TrimSpace(language), "\"")
		if language == "" || len(language) > 32 {
			return LocalSpecification{}, ErrOfficialSnapshotInvalid
		}
		s.Languages = append(s.Languages, language)
	}
	s.Weight.Unit, s.Dimension.Unit = row[4], row[7]
	if s.Weight.Number, err = parseNumberRule(row[5], row[6]); err != nil {
		return LocalSpecification{}, err
	}
	if s.Dimension.Number, err = parseNumberRule(row[8], row[9]); err != nil {
		return LocalSpecification{}, err
	}
	if s.RetailPrice, err = parseNumberRule(row[12], row[13]); err != nil {
		return LocalSpecification{}, err
	}
	if s.BasePrice, err = parseNumberRule(row[12], row[14]); err != nil {
		return LocalSpecification{}, err
	}
	if row[1] != "US" && row[1] != "EU" && row[1] != "GLO" {
		return LocalSpecification{}, ErrOfficialSnapshotInvalid
	}
	if row[4] != "lb" && row[4] != "g" || row[7] != "in" && row[7] != "cm" || len(row[10]) != 3 || len(row[11]) != 3 {
		return LocalSpecification{}, ErrOfficialSnapshotInvalid
	}
	return s, nil
}
