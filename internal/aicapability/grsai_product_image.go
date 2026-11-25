package aicapability

const (
	GRSaiProductImageModel          = "gpt-image-2.6"
	GRSaiProductImageProtocol       = "grsai-product-image-async-v1"
	GRSaiProductImageBaseURL        = "https://grsaiapi.com"
	GRSaiProductImageChinaBaseURL   = "https://grsai.dakka.com.cn"
	GRSaiProductImageMaxSourceBytes = 3 << 20
)

func GRSaiProductImageRegion(baseURL string) string {
	switch baseURL {
	case GRSaiProductImageBaseURL:
		return "global"
	case GRSaiProductImageChinaBaseURL:
		return "cn"
	default:
		return ""
	}
}

func ValidProductImageTaskID(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}
	for _, ch := range id {
		if (ch < 'A' || ch > 'Z') && (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' && ch != '_' {
			return false
		}
	}
	return true
}
