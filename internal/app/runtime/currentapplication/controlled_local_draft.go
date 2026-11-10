package currentapplication

import (
	"errors"
	listingapp "task-processor/internal/app/listing"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
)

type ControlledLocalDraftConfig struct {
	Enabled            bool           `json:"enabled"`
	Acknowledgment     string         `json:"acknowledgment"`
	OrganizationID     string         `json:"organizationId"`
	ActorID            string         `json:"actorId"`
	MemberID           string         `json:"memberId"`
	SourceID           string         `json:"sourceId"`
	ProductKey         string         `json:"productKey"`
	CompletionRecordID string         `json:"completionRecordId"`
	CompletionRevision int64          `json:"completionRevision"`
	StoreID            string         `json:"storeId"`
	FixtureVersion     string         `json:"fixtureVersion"`
	FixtureHash        string         `json:"fixtureHash"`
	AssetDatabase      DatabaseConfig `json:"assetDatabase"`
}

func (t ControlledLocalDraftConfig) Policy() record.LocalTrialPolicy {
	return record.LocalTrialPolicy{Scope: collection.Scope{OrganizationID: t.OrganizationID, ActorID: t.ActorID, MemberID: t.MemberID}, Binding: record.LocalTrial{ContractVersion: 1, TrialID: "issue1158-local-draft-v1", SourceID: t.SourceID, CompletionDraft: record.CompletionDraftSelection{RecordID: t.CompletionRecordID, Revision: t.CompletionRevision}, StoreID: t.StoreID, FixtureVersion: t.FixtureVersion, FixtureHash: t.FixtureHash}, ProductKey: t.ProductKey, ProductVersion: 1, Enabled: t.Enabled}
}

func (c *Config) validateControlledLocalDraft() error {
	t := c.ControlledLocalDraft
	if t == nil {
		return nil
	}
	if t.Acknowledgment != "CONTROLLED_LOCAL_DRAFT_NO_EXECUTION" || t.OrganizationID != "394513989062623234" || t.ActorID != "394513839896395778" || t.MemberID != "394513991663091714" || t.SourceID != "41cb802a-fa17-523e-9444-12044f47e5dc" || t.ProductKey != "own-27ef901c-30e0-57f9-81d9-56032b0efa61" || t.CompletionRecordID != "f7853e11-353a-5214-a7c7-75a8ded6e278" || t.CompletionRevision != 1 || !collection.ValidID(t.StoreID) || t.StoreID == "b6e4504f-c3db-4e1b-9c5f-2605dda4aea1" || t.FixtureVersion != "issue1158-local-draft-v1" || t.FixtureHash != collection.Digest(listingapp.ControlledLocalRules()) {
		return errors.New("controlled local draft requires the admitted fixed scope/source/R1 and separate synthetic store/fixture")
	}
	if c.SupplyChain != nil || c.PrivateDraftTrial != nil || c.SheinOfflineRecords != nil || c.LocalTrial != nil || !c.ProductCollections || !c.SupplyCompletionDrafts || c.ProductAcquisitionDatabase == nil || c.StoreCenter == nil || !c.StoreCenter.Enabled || c.Identity.TenantDirectoryToken == "" {
		return errors.New("controlled local draft requires current owners and excludes other Target serving modes")
	}
	issuer, err := validateLoopbackURL("issuerURL", c.Identity.IssuerURL)
	if err != nil || issuer.Scheme != "https" {
		return errors.New("controlled local draft requires literal loopback HTTPS identity")
	}
	if err = t.AssetDatabase.validate("controlledLocalDraft.assetDatabase"); err != nil {
		return err
	}
	if t.AssetDatabase.User != "supply_asset_runtime" || t.AssetDatabase.MaxConnections > 8 {
		return errors.New("controlled local draft requires bounded canonical Asset reader")
	}
	if err = c.validateAssetOwnerIsolation(t.AssetDatabase); err != nil {
		return err
	}
	if c.SourceMaterials != nil && !sameDatabaseTarget(t.AssetDatabase, c.SourceMaterials.AssetDatabase) || c.ImageAgent != nil && !sameDatabaseTarget(t.AssetDatabase, c.ImageAgent.Database) || c.ProductAgent != nil && c.ProductAgent.Enabled && !c.ProductAgent.ContentTitleOnly && !c.ProductAgent.ContentOnly && !sameDatabaseTarget(t.AssetDatabase, c.ProductAgent.AssetDatabase) {
		return errors.New("controlled local draft must reuse the current Asset owner")
	}
	return nil
}
