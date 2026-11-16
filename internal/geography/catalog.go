package geography

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
)

//go:embed catalog.json.gz
var snapshot []byte

//go:embed china.json
var chinaSnapshot []byte

// Direct identifies the explicit no-subdivision group in a country that also has provinces.
const Direct = "direct"

var ErrInvalidSelection = errors.New("invalid region catalog selection")

type Option struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Province struct {
	Option
	Cities []Option `json:"cities"`
}

type Country struct {
	Option
	Provinces []Province `json:"provinces"`
	Cities    []Option   `json:"cities"`
}

type Labels struct {
	Country  string
	Province string
	City     string
}

type Catalog struct {
	version   string
	countries []Option
	byID      map[string]*Country
}

var loadCatalog = sync.OnceValues(func() (*Catalog, error) {
	reader, err := gzip.NewReader(bytes.NewReader(snapshot))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	var data struct {
		Version   string    `json:"version"`
		Countries []Country `json:"countries"`
	}
	if err := json.NewDecoder(reader).Decode(&data); err != nil {
		return nil, err
	}
	if data.Version == "" || len(data.Countries) == 0 {
		return nil, errors.New("empty region catalog")
	}
	var china Country
	if err := json.Unmarshal(chinaSnapshot, &china); err != nil {
		return nil, err
	}
	if china.ID != "CN" || len(china.Provinces) == 0 {
		return nil, errors.New("empty China region catalog")
	}
	digest := sha256.Sum256([]byte(data.Version + "\n" + string(bytes.TrimSpace(chinaSnapshot))))
	catalog := &Catalog{version: "gn-" + hex.EncodeToString(digest[:8]), byID: make(map[string]*Country, len(data.Countries))}
	for i := range data.Countries {
		country := &data.Countries[i]
		if country.ID == china.ID {
			if country.Name != china.Name || len(country.Provinces) != 0 || len(country.Cities) != 0 {
				return nil, errors.New("invalid China country metadata")
			}
			country = &china
		}
		catalog.countries = append(catalog.countries, country.Option)
		catalog.byID[country.ID] = country
	}
	if catalog.Country(china.ID) == nil {
		return nil, errors.New("missing China country metadata")
	}
	return catalog, nil
})

// Bundled returns the immutable catalog shipped with this version of the application.
func Bundled() (*Catalog, error) { return loadCatalog() }

func (c *Catalog) Version() string { return c.version }

// Countries and Country return read-only views of the embedded reference data.
func (c *Catalog) Countries() []Option        { return c.countries }
func (c *Catalog) Country(id string) *Country { return c.byID[id] }

func (c *Catalog) Resolve(version, countryID, provinceID, cityID string) (Labels, error) {
	if version != c.version {
		return Labels{}, ErrInvalidSelection
	}
	if countryID == "" && provinceID == "" && cityID == "" {
		return Labels{}, nil
	}
	country := c.Country(countryID)
	if country == nil {
		return Labels{}, ErrInvalidSelection
	}
	labels := Labels{Country: country.Name}
	cities := country.Cities
	if len(country.Provinces) > 0 {
		if provinceID == Direct && len(cities) > 0 {
			// Direct cities have no persisted province label.
		} else {
			var matched bool
			for _, province := range country.Provinces {
				if province.ID == provinceID {
					labels.Province, cities, matched = province.Name, province.Cities, true
					break
				}
			}
			if !matched {
				return Labels{}, ErrInvalidSelection
			}
		}
	} else if provinceID != "" {
		return Labels{}, ErrInvalidSelection
	}
	if len(cities) == 0 && cityID == "" {
		return labels, nil
	}
	for _, city := range cities {
		if city.ID == cityID {
			labels.City = city.Name
			return labels, nil
		}
	}
	return Labels{}, ErrInvalidSelection
}
