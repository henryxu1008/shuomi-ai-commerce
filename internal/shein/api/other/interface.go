package other

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// FlexibleString can unmarshal both string and number values from JSON
type FlexibleString string

// UnmarshalJSON implements json.Unmarshaler interface
func (fs *FlexibleString) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*fs = FlexibleString(s)
		return nil
	}

	// If that fails, try as number
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*fs = FlexibleString(n.String())
		return nil
	}

	// If both fail, try as int64
	var i int64
	if err := json.Unmarshal(data, &i); err == nil {
		*fs = FlexibleString(strconv.FormatInt(i, 10))
		return nil
	}

	// If both fail, try as float64
	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		*fs = FlexibleString(fmt.Sprintf("%.0f", f))
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into FlexibleString", string(data))
}

// String returns the string representation
func (fs FlexibleString) String() string {
	return string(fs)
}

type SupplierOperateInfoResponse struct {
	Code string              `json:"code"`
	Msg  string              `json:"msg"`
	Info SupplierOperateInfo `json:"info"`
	BBL  *string             `json:"bbl"`
}

// SupplierOperateInfo SSO供应商操作信息
type SupplierOperateInfo struct {
	SupplierID            int64           `json:"supplierId"`
	StoreID               int64           `json:"storeId"`
	StoreTitle            *string         `json:"storeTitle"`
	StoreLogo             string          `json:"storeLogo"`
	CooperationModelCode  string          `json:"cooperationModelCode"`
	CooperationModelTitle string          `json:"cooperationModelTitle"`
	CurrentLevel          string          `json:"currentLevel"`
	UserName              string          `json:"userName"`
	UlpName               *string         `json:"ulpName"`
	UlpEnName             *string         `json:"ulpEnName"`
	UlpEmplid             *string         `json:"ulpEmplid"`
	IsUlpLogin            int             `json:"isUlpLogin"`
	IsChildren            int             `json:"isChildren"`
	RoleList              []string        `json:"roleList"`
	OperationType         *FlexibleString `json:"operationType"`
	OperationContact      *string         `json:"operationContact"`
	OperationContactEn    *string         `json:"operationContactEn"`
	OperationContactEnDpt *string         `json:"operationContactEnDpt"`
	IsWxBind              int             `json:"isWxBind"`
	JoinWechat            int             `json:"joinWechat"`
	IsShowOperationInfo   int             `json:"isShowOperationInfo"`
	OperationGroupName    *string         `json:"operationGroupName"`
	IsSimplePlatform      bool            `json:"isSimplePlatform"`
	IsShowWecomQrCode     bool            `json:"isShowWecomQrCode"`
}
