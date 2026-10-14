package goods

// Source bytes and a platform re-encoded artifact have separate digests. This
// fact proves content, not source approval or indefinite URL availability.
type ImageArtifact struct {
	URL         string `json:"url"`
	ContentHash string `json:"contentHash"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Bytes       int64  `json:"bytes"`
	MediaType   string `json:"mediaType"`
}

func ValidImageArtifact(a ImageArtifact) bool {
	return a.Bytes <= 2<<20 && ValidImageObservation(ImageObservation{AssetID: "official-artifact", SourceURL: a.URL, CatID: 1, Usage: 2, ContentHash: a.ContentHash, Width: a.Width, Height: a.Height, Bytes: a.Bytes, MediaType: a.MediaType})
}
