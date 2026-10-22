package sheinlogin

import (
	"task-processor/internal/storecenter"
	"time"
)

type FailureSummary struct {
	ErrorCode            string    `json:"error_code,omitempty"`
	ErrorMessage         string    `json:"error_message,omitempty"`
	PageState            string    `json:"page_state,omitempty"`
	ActionKey            string    `json:"action_key,omitempty"`
	ActionMessage        string    `json:"action_message,omitempty"`
	ArtifactPath         string    `json:"artifact_path,omitempty"`
	CapturedAt           time.Time `json:"captured_at,omitempty"`
	Stage                string    `json:"stage,omitempty"`
	URL                  string    `json:"url,omitempty"`
	Title                string    `json:"title,omitempty"`
	LoginError           string    `json:"login_error,omitempty"`
	WaitingForVerifyCode bool      `json:"waiting_for_verify_code,omitempty"`
}

type FailureDetail struct {
	ErrorCode              string           `json:"error_code,omitempty"`
	ErrorMessage           string           `json:"error_message,omitempty"`
	PageState              string           `json:"page_state,omitempty"`
	ActionKey              string           `json:"action_key,omitempty"`
	ActionMessage          string           `json:"action_message,omitempty"`
	ArtifactPath           string           `json:"artifact_path,omitempty"`
	CapturedAt             time.Time        `json:"captured_at,omitempty"`
	Stage                  string           `json:"stage,omitempty"`
	URL                    string           `json:"url,omitempty"`
	Title                  string           `json:"title,omitempty"`
	LoginError             string           `json:"login_error,omitempty"`
	WaitingForVerifyCode   bool             `json:"waiting_for_verify_code,omitempty"`
	OnLoginPage            bool             `json:"on_login_page,omitempty"`
	RequestFailureModal    bool             `json:"request_failure_modal,omitempty"`
	LoginFormVisible       bool             `json:"login_form_visible,omitempty"`
	SellerHubVisible       bool             `json:"seller_hub_visible,omitempty"`
	VerificationVisible    bool             `json:"verification_visible,omitempty"`
	PermissionVisible      bool             `json:"permission_visible,omitempty"`
	AgreementVisible       bool             `json:"agreement_visible,omitempty"`
	CredentialErrorVisible bool             `json:"credential_error_visible,omitempty"`
	BodyText               string           `json:"body_text,omitempty"`
	SelectorStates         map[string]bool  `json:"selector_states,omitempty"`
	NetworkPayloads        []map[string]any `json:"network_payloads,omitempty"`
}

type RecommendedAction struct {
	Key     string `json:"key,omitempty"`
	Message string `json:"message,omitempty"`
}

// Account is loaded only inside the executor from a fresh private binding.
// Login material is never serialized into attempts or public output.
type Account struct {
	Scope           storecenter.SheinLoginScope `json:"-"`
	OrganizationID  string                      `json:"-"`
	StoreID         string                      `json:"-"`
	BindingRevision string                      `json:"-"`
	Generation      int64                       `json:"-"`
	Username        string                      `json:"-"`
	Password        string                      `json:"-"`
	LoginURL        string                      `json:"-"`
	Proxy           string                      `json:"-"`
}

type LoginAttemptStatus string

const (
	LoginAttemptQueued            LoginAttemptStatus = "queued"
	LoginAttemptLaunching         LoginAttemptStatus = "launching"
	LoginAttemptWaitingVerifyCode LoginAttemptStatus = "waiting_verify_code"
	LoginAttemptSucceeded         LoginAttemptStatus = "succeeded"
	LoginAttemptFailed            LoginAttemptStatus = "failed"
	LoginAttemptCancelled         LoginAttemptStatus = "cancelled"
	LoginAttemptInterrupted       LoginAttemptStatus = "interrupted"
)

func (s LoginAttemptStatus) IsActive() bool {
	return s == LoginAttemptQueued || s == LoginAttemptLaunching || s == LoginAttemptWaitingVerifyCode
}

type LoginAttempt struct {
	ID                   string                      `json:"id"`
	Scope                storecenter.SheinLoginScope `json:"scope"`
	ActorID              string                      `json:"actor_id"`
	MemberID             string                      `json:"member_id"`
	Fingerprint          string                      `json:"fingerprint"`
	ForceLogin           bool                        `json:"force_login"`
	Status               LoginAttemptStatus          `json:"status"`
	ErrorCode            string                      `json:"error_code,omitempty"`
	SyncStatus           string                      `json:"sync_status"`
	MerchantID           string                      `json:"merchant_id,omitempty"`
	ExpectedStoreVersion int64                       `json:"expected_store_version,omitempty"`
	OwnerNonce           string                      `json:"owner_nonce,omitempty"`
	CreatedAt            time.Time                   `json:"created_at"`
	CompletedAt          *time.Time                  `json:"completed_at,omitempty"`
}

// Result is the only CLI response. It deliberately excludes credentials,
// provider bodies, cookies, verification codes and private artifacts.
type Result struct {
	OrganizationID string `json:"organization_id"`
	StoreID        string `json:"store_id"`
	AttemptID      string `json:"attempt_id,omitempty"`
	Status         string `json:"status"`
	SyncStatus     string `json:"sync_status,omitempty"`
	ErrorCode      string `json:"error_code,omitempty"`
	Generation     int64  `json:"generation"`
}

func (a LoginAttempt) Result() Result {
	return Result{a.Scope.OrganizationID, a.Scope.StoreID, a.ID, string(a.Status), a.SyncStatus, a.ErrorCode, a.Scope.Generation}
}

type WarehouseOption struct {
	WarehouseCode   string   `json:"warehouse_code"`
	WarehouseName   string   `json:"warehouse_name"`
	SaleCountryList []string `json:"sale_country_list"`
	WarehouseType   int      `json:"warehouse_type"`
}
