package materials

import (
	"testing"

	"github.com/stretchr/testify/require"
	record "task-processor/internal/listing/record/target"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/collection"
)

func TestMaterialCommitKeepsExactProposalSourceRulesAssetsAndCAS(t *testing.T) {
	i := materialInspection(t)
	p, err := ParseProposal(proposalFor(t, i, `"brandCode":"known-brand"`).Payload)
	require.NoError(t, err)
	preview, err := Preview(i, p)
	require.NoError(t, err)
	valid := func() record.TargetCommitFacts {
		r := i.Base
		r.ID, r.Revision, r.Input = "new-record", i.Base.Revision+1, preview.Input
		return record.TargetCommitFacts{Record: r, Rules: i.Rules, Inventory: i.Inventory}
	}
	require.NoError(t, VerifyCommit(i, p, valid()))
	for _, change := range []func(*record.TargetCommitFacts){
		func(f *record.TargetCommitFacts) {
			f.Record.Input.Draft.Product.Names = []model.LanguageContent{{Language: "en", Name: "unapproved content"}}
		},
		func(f *record.TargetCommitFacts) { f.Record.Input.ExpectedRevision++ },
		func(f *record.TargetCommitFacts) { f.Record.Merchant.StoreVersion++ },
		func(f *record.TargetCommitFacts) { f.Record.ProductHash = "changed" },
		func(f *record.TargetCommitFacts) { f.Inventory.Assets = nil },
		func(f *record.TargetCommitFacts) {
			f.Rules.Brands = nil
			f.Record.RulesHash = collection.Digest(f.Rules)
		},
	} {
		f := valid()
		change(&f)
		require.ErrorIs(t, VerifyCommit(i, p, f), ErrInspectionChanged)
	}
}

func TestMaterialCategoryCommitUsesActualNewRulesAndOriginalDisclosedCatalog(t *testing.T) {
	i := materialInspection(t)
	p, err := ParseProposal(proposalFor(t, i, `"categoryId":13`).Payload)
	require.NoError(t, err)
	preview, err := Preview(i, p)
	require.NoError(t, err)
	f := record.TargetCommitFacts{Record: i.Base, Rules: i.Rules, Inventory: i.Inventory}
	f.Record.ID, f.Record.Revision, f.Record.Input = "new-record", i.Base.Revision+1, preview.Input
	f.Rules.Attributes = model.AttributeTemplate{ProductTypeID: 999}
	f.Record.RulesHash = collection.Digest(f.Rules)
	require.NoError(t, VerifyCommit(i, p, f), "actual new-category rules are consumed by the existing owner")
	f.Rules.Categories = nil
	f.Record.RulesHash = collection.Digest(f.Rules)
	require.ErrorIs(t, VerifyCommit(i, p, f), ErrInspectionChanged)
}
