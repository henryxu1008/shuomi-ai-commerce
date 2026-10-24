// Package product 产品响应数据结构
package product

// BrandListResponse 品牌列表响应
type BrandListResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Info struct {
		Data []BrandItem `json:"data"`
		Meta struct {
			Count     int `json:"count"`
			CustomObj any `json:"customObj"`
		} `json:"meta"`
	} `json:"info"`
	BBL any `json:"bbl"`
}

// BrandItem 品牌项
type BrandItem struct {
	BrandCode   string `json:"brand_code"`
	BrandName   string `json:"brand_name"`
	BrandNameEn string `json:"brand_name_en"`
}
