package model

type LocalImageUpload struct {
	FileURL string `json:"fileUrl"`
	CatID   int64  `json:"catId"`
	Usage   int64  `json:"usage"`
}

type LocalImageUploaded struct {
	Images []LocalUploadedImage `json:"images"`
}
type LocalUploadedImage struct {
	URL string `json:"url"`
}
