package assetpublication

import (
	"context"
	"reflect"
	"strings"
	"task-processor/internal/imageagent"
	productasset "task-processor/internal/product/asset"
)

type ProjectionSource interface {
	GetProjection(context.Context, imageagent.RunScope) (imageagent.RunProjection, error)
}

func publicationAcknowledgement(commit productasset.ApprovalCommit, runID string, revision int64, digest string, receipt productasset.ApprovalReceipt) imageagent.PublicationAcknowledgement {
	return imageagent.PublicationAcknowledgement{
		ProductKey: commit.ProductKey, RunID: runID, PlanRevision: revision,
		ResultDigest: digest, ActionID: receipt.ActionID, AssetIDs: append([]string(nil), receipt.AssetIDs...),
	}
}
func canonical(value string) bool {
	return value != "" && strings.TrimSpace(value) == value
}
func nilValue(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
func sourceSnapshotVersion(catalog imageagent.AssetCatalog) uint64 {
	return catalog.ProductContext.SourceSnapshotVersion
}
func approvedRole(role imageagent.SlotRole) (productasset.Role, error) {
	if err := imageagent.ValidateSlotRole(role); err != nil {
		return "", err
	}
	if role == imageagent.SlotRoleMain {
		return productasset.RoleMain, nil
	}
	return productasset.RoleGallery, nil
}
