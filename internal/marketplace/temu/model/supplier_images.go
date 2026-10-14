package model

// E43 is a different protocol from the local image V2 URL/usage request.
// Image is transient base64 prepared from the exact approved source bytes.
type SupplierImageUpload struct {
	Image        string               `json:"image"`
	BusinessType int64                `json:"imageBizType"`
	Options      SupplierImageOptions `json:"options"`
}

type SupplierImageOptions struct {
	CategoryID       int64 `json:"cateId"`
	IntelligenceCrop bool  `json:"doIntelligenceCrop"`
	Boost            bool  `json:"boost"`
	SizeMode         int64 `json:"sizeMode"`
}

type SupplierImageUploaded struct {
	URLs     []string `json:"urls"`
	ImageURL string   `json:"imageUrl"`
	URL      string   `json:"url"`
}
