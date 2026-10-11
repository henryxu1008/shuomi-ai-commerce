package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"task-processor/internal/httproute"
	"task-processor/internal/storecenter"
)

type CatalogStoreResponse struct {
	StoreResponse
	Profile   storecenter.StoreProfile   `json:"profile"`
	Signals   storecenter.CatalogSignals `json:"signals"`
	Attention string                     `json:"attention"`
}

func rejectStoreCenterReadBody(c *gin.Context) bool {
	if c.Request.ContentLength == 0 && len(c.Request.TransferEncoding) == 0 {
		return false
	}
	httproute.RejectUnreadRequestBody(c)
	writeInvalid(c, "body", "not_allowed")
	return true
}

func (h *Handler) StoreSummary(c *gin.Context) {
	identity, ok := authoritativeIdentity(c)
	if !ok {
		return
	}
	if rejectStoreCenterReadBody(c) {
		return
	}
	if requireNoQuery(c.Request.URL.RawQuery) != nil {
		writeInvalid(c, "request", "invalid")
		return
	}
	id, err := canonicalUUID(c.Param("store_id"))
	if err != nil {
		writeInvalid(c, "store_id", "invalid")
		return
	}
	item, err := h.catalog.Get(c.Request.Context(), identity.EffectiveOrganizationID, id)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	base, err := storeResponse(&item.Store, item.Signals.Connection, identity.EffectiveOrganizationID, id)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	storeCenterResponse(c, CatalogStoreResponse{StoreResponse: base, Profile: item.Profile, Signals: item.Signals, Attention: item.Attention})
}

func (h *Handler) catalogList(c *gin.Context) {
	if rejectStoreCenterReadBody(c) {
		return
	}
	identity, ok := authoritativeIdentity(c)
	if !ok {
		return
	}
	query, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil {
		writeInvalid(c, "query", "invalid")
		return
	}
	base := url.Values{}
	allowed := map[string]bool{"q": true, "region": true, "applicationType": true, "groupId": true, "attention": true}
	for key, values := range query {
		if len(values) != 1 {
			writeInvalid(c, key, "invalid")
			return
		}
		if !allowed[key] {
			base[key] = values
		}
	}
	basic, field, err := parseListRequest(base.Encode())
	if err != nil {
		writeInvalid(c, field, "invalid")
		return
	}
	result, err := h.catalog.List(c.Request.Context(), storecenter.CatalogRequest{OrganizationID: identity.EffectiveOrganizationID, Page: basic.Page, PageSize: basic.PageSize, Platform: basic.Platform, Status: basic.Status, Search: query.Get("q"), Region: query.Get("region"), ApplicationType: query.Get("applicationType"), GroupID: query.Get("groupId"), Attention: query.Get("attention")})
	if errors.Is(err, storecenter.ErrCatalogSummaryIncomplete) {
		writeError(c, http.StatusServiceUnavailable, "CATALOG_SUMMARY_UNAVAILABLE", "当前摘要不足，无法完整应用此筛选", nil)
		return
	}
	if err != nil {
		writeStoreError(c, err)
		return
	}
	items := []CatalogStoreResponse{}
	for _, item := range result.Items {
		response, err := storeResponse(&item.Store, item.Signals.Connection, identity.EffectiveOrganizationID, item.Store.ID())
		if err != nil {
			writeStoreError(c, err)
			return
		}
		items = append(items, CatalogStoreResponse{StoreResponse: response, Profile: item.Profile, Signals: item.Signals, Attention: item.Attention})
	}
	payload := struct {
		Items      []CatalogStoreResponse     `json:"items"`
		Pagination PaginationResponse         `json:"pagination"`
		Metrics    storecenter.CatalogMetrics `json:"metrics"`
		Facets     storecenter.CatalogFacets  `json:"facets"`
	}{items, PaginationResponse{Page: result.Page, PageSize: result.PageSize, Total: result.Total}, result.Metrics, result.Facets}
	data, err := json.Marshal(gin.H{"schemaVersion": "store-center-v1", "organizationId": identity.EffectiveOrganizationID, "userId": identity.UserID, "data": payload})
	if err != nil || len(data) > 2<<20 {
		writeStoreError(c, storecenter.ErrDependencyUnavailable)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", data)
}
func (h *Handler) Groups(c *gin.Context) {
	identity, ok := authoritativeIdentity(c)
	if !ok {
		return
	}
	if rejectStoreCenterReadBody(c) {
		return
	}
	if requireNoQuery(c.Request.URL.RawQuery) != nil {
		writeInvalid(c, "request", "invalid")
		return
	}
	result, err := h.metadata.Groups(c.Request.Context(), identity.EffectiveOrganizationID)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	storeCenterResponse(c, gin.H{"items": result})
}
func (h *Handler) Profile(c *gin.Context) {
	identity, ok := authoritativeIdentity(c)
	if !ok {
		return
	}
	if rejectStoreCenterReadBody(c) {
		return
	}
	if requireNoQuery(c.Request.URL.RawQuery) != nil {
		writeInvalid(c, "request", "invalid")
		return
	}
	id, err := canonicalUUID(c.Param("store_id"))
	if err != nil {
		writeInvalid(c, "store_id", "invalid")
		return
	}
	result, err := h.metadata.Profile(c.Request.Context(), identity.EffectiveOrganizationID, id)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	storeCenterResponse(c, result)
}
func (h *Handler) MetadataReceipt(c *gin.Context) {
	identity, ok := authoritativeIdentity(c)
	if !ok {
		return
	}
	if rejectStoreCenterReadBody(c) {
		return
	}
	if requireNoQuery(c.Request.URL.RawQuery) != nil {
		writeInvalid(c, "request", "invalid")
		return
	}
	result, err := h.metadata.Receipt(c.Request.Context(), identity.EffectiveOrganizationID, c.Param("operation_id"))
	if err != nil {
		writeStoreError(c, err)
		return
	}
	storeCenterResponse(c, result)
}
func (h *Handler) MetadataCommand(c *gin.Context) {
	identity, ok := authoritativeIdentity(c)
	if !ok {
		return
	}
	if requireNoQuery(c.Request.URL.RawQuery) != nil {
		writeInvalid(c, "query", "invalid")
		return
	}
	key, err := requiredCanonicalUUIDHeader(c.Request, "Idempotency-Key")
	if err != nil {
		writeInvalid(c, "Idempotency-Key", "invalid")
		return
	}
	raw, err := readBoundedBody(c.Request.Body)
	if err != nil || !utf8.Valid(raw) {
		writeInvalid(c, "body", "invalid")
		return
	}
	if err = uniqueJSON(json.NewDecoder(bytes.NewReader(raw))); err != nil {
		writeInvalid(c, "body", "invalid")
		return
	}
	if err = explicitMetadataFields(raw); err != nil {
		writeInvalid(c, "body", "invalid")
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var command storecenter.MetadataCommand
	if err = decoder.Decode(&command); err != nil {
		writeInvalid(c, "body", "invalid")
		return
	}
	command.OperationID = key
	result, err := h.metadata.Execute(c.Request.Context(), identity.EffectiveOrganizationID, command)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	storeCenterResponse(c, result)
}

func explicitMetadataFields(raw []byte) error {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return errors.New("metadata object required")
	}
	for _, key := range []string{"action", "name", "groupId", "groupVersion", "stores", "responsibleMemberId", "lowStockThreshold"} {
		value, ok := fields[key]
		if !ok || (key == "action" || key == "name" || key == "groupVersion" || key == "stores") && bytes.Equal(value, []byte("null")) {
			return errors.New("explicit intent required")
		}
	}
	var targets []map[string]json.RawMessage
	if json.Unmarshal(fields["stores"], &targets) != nil {
		return errors.New("invalid targets")
	}
	for _, target := range targets {
		for _, key := range []string{"storeId", "profileVersion"} {
			value, ok := target[key]
			if !ok || bytes.Equal(value, []byte("null")) {
				return errors.New("explicit target version required")
			}
		}
	}
	return nil
}

func storeCenterResponse(c *gin.Context, data any) {
	identity, ok := authoritativeIdentity(c)
	if !ok {
		return
	}
	encoded, err := json.Marshal(gin.H{"schemaVersion": "store-center-v1", "organizationId": identity.EffectiveOrganizationID, "userId": identity.UserID, "data": data})
	if err != nil || len(encoded) > 2<<20 {
		writeStoreError(c, storecenter.ErrDependencyUnavailable)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", encoded)
}

// Recursive duplicate detection covers profile targets too; encoding/json alone
// accepts a later duplicate and could change the signed command intent.
func uniqueJSON(decoder *json.Decoder) error {
	var visit func() error
	visit = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		if delim == '{' {
			seen := map[string]bool{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return errors.New("duplicate JSON field")
				}
				seen[name] = true
				if err = visit(); err != nil {
					return err
				}
			}
		} else if delim == '[' {
			for decoder.More() {
				if err = visit(); err != nil {
					return err
				}
			}
		} else {
			return errors.New("invalid JSON delimiter")
		}
		_, err = decoder.Token()
		return err
	}
	if err := visit(); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}
