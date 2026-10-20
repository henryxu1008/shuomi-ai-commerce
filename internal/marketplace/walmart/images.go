package walmart

type ImageObservation struct {
	AssetID     string `json:"assetId"`
	URL         string `json:"url"`
	ContentHash string `json:"contentHash"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Bytes       int64  `json:"bytes"`
	MediaType   string `json:"mediaType"`
}
