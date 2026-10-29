package membership

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"task-processor/internal/authidentity"
	domain "task-processor/internal/organization/membership"
)

const authorizationService = "/zitadel.authorization.v2.AuthorizationService/"

func (c *Client) ReadAuthorizationRevision(ctx context.Context, organization, id string) (domain.NativeAuthorizationRevision, error) {
	if c == nil || !authidentity.IsBoundedIdentifier(organization) || !authidentity.IsBoundedIdentifier(id) {
		return domain.NativeAuthorizationRevision{}, domain.ErrInvalidRequest
	}
	if err := c.requirePermission(ctx, organization, "user.grant.read"); err != nil {
		return domain.NativeAuthorizationRevision{}, err
	}
	raw, err := c.requestJSON(ctx, http.MethodPost, authorizationService+"GetAuthorizationRevision", map[string]any{"authorizationId": id, "organizationId": organization, "projectId": c.project}, false, organization)
	if err != nil {
		return domain.NativeAuthorizationRevision{}, err
	}
	var response struct {
		Revision domain.NativeAuthorizationRevision `json:"revision"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&response) != nil || !response.Revision.Valid() || response.Revision.AuthorizationID != id || response.Revision.OrganizationID != organization || response.Revision.ProjectID != c.project {
		return domain.NativeAuthorizationRevision{}, domain.ErrInvalidResponse
	}
	if _, err := time.Parse(time.RFC3339Nano, response.Revision.ChangedAt); err != nil {
		return domain.NativeAuthorizationRevision{}, domain.ErrInvalidResponse
	}
	return response.Revision, nil
}

func (w *Writer) writeConditionalRole(ctx context.Context, op domain.Operation) (domain.Acknowledgment, error) {
	if !domain.ConditionalRoleOperationValid(op) {
		return domain.Acknowledgment{}, domain.ErrInvalidRequest
	}
	native := op.ConditionalRole.Native
	body := map[string]any{"authorizationId": op.AuthorizationID, "organizationId": op.Scope.OrganizationID, "projectId": op.Scope.ProjectID, "projectGrantId": native.ProjectGrantID, "expectedSequence": native.NativeSequence, "roleKey": op.Role, "direction": op.ConditionalRole.Direction, "dispatchId": op.DispatchID}
	raw, err := w.write.requestJSON(ctx, http.MethodPost, authorizationService+"ChangeAuthorizationRoleConditional", body, false, op.Scope.OrganizationID)
	if err != nil {
		return domain.Acknowledgment{}, err
	}
	var response struct {
		Evidence   *domain.RoleWriteEvidence `json:"evidence"`
		ChangeDate string                    `json:"changeDate"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&response) != nil || response.Evidence == nil || !response.Evidence.MatchesOperation(op) {
		return domain.Acknowledgment{}, domain.ErrInvalidResponse
	}
	e := response.Evidence
	if e.Result == domain.RoleWriteNoEffect && e.NativeSequence == "" && response.ChangeDate == "" {
		return domain.Acknowledgment{}, &domain.ProviderNoEffectError{Evidence: *e}
	}
	if e.Result != domain.RoleWriteApplied {
		return domain.Acknowledgment{}, domain.ErrInvalidResponse
	}
	if _, err := time.Parse(time.RFC3339Nano, response.ChangeDate); err != nil {
		return domain.Acknowledgment{}, domain.ErrInvalidResponse
	}
	ack := domain.Acknowledgment{ID: op.AuthorizationID, At: response.ChangeDate, ConditionalRole: e}
	if _, err := domain.Transition(op, domain.OperationChange{Event: domain.EventAcknowledge, DispatchID: op.DispatchID, Acknowledgment: &ack}); err != nil {
		return domain.Acknowledgment{}, domain.ErrInvalidResponse
	}
	return ack, nil
}
