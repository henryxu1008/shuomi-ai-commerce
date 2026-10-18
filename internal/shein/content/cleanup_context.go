package content

import (
	"context"

	"task-processor/internal/shein/api/product"
	"task-processor/internal/shein/authorizedbrand"
)

// CleanupContext supplies only the values needed for content cleanup.
type CleanupContext struct {
	Context         context.Context
	AuthorizedBrand *authorizedbrand.Resolved
	SourceBrand     string
	ProductData     *product.Product
}

// ValidationResult preserves platform messages across language and SKC scopes.
type ValidationResult struct {
	Form                    string                           `json:"form"`
	FormName                string                           `json:"form_name"`
	Messages                []string                         `json:"messages"`
	Module                  string                           `json:"module"`
	OtherLanguageMessageMap map[string][]string              `json:"other_language_message_map"`
	SkcErrorMessageMap      map[string]SKCValidationMessages `json:"skc_error_message_map"`
}

type SKCValidationMessages struct {
	Messages                []string            `json:"messages"`
	OtherLanguageMessageMap map[string][]string `json:"otherLanguageMessageMap"`
}
