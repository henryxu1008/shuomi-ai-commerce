package offlinepackage

import (
	"reflect"
	"sort"
	"task-processor/internal/marketplace/shein/offlinepackage/previewdata"
	"testing"
)

func TestClonePackageForPersistencePreservesCanonicalFieldsAndPrivateAssignmentsWithoutSharingState(t *testing.T) {
	draft := &RequestDraft{SpuName: "original-draft"}
	preview := &previewdata.Product{SPUName: "original-preview"}
	finalDraft := &FinalDraft{MainImageURL: "https://img.example/original.jpg"}
	valueID := 701
	resolution := &SaleAttributeResolution{
		SKCValueAssignments: map[string]ResolvedSaleAttribute{"blue": {Value: "Blue", AttributeValueID: &valueID}},
		SKUValueAssignments: map[string]ResolvedSaleAttribute{"xl": {Value: "XL", AttributeValueID: &valueID}},
	}
	pkg := &Package{
		DraftPayload:            draft,
		PreviewPayload:          preview,
		FinalSubmissionDraft:    finalDraft,
		SaleAttributeResolution: resolution,
		Metadata:                map[string]string{"owner": "source"},
	}

	cloned, err := ClonePackageForPersistence(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if cloned.DraftPayload == nil || cloned.DraftPayload == pkg.DraftPayload {
		t.Fatal("draft canonical field is missing or shares source state")
	}
	if cloned.PreviewPayload == nil || cloned.PreviewPayload == pkg.PreviewPayload {
		t.Fatal("preview canonical field is missing or shares source state")
	}
	if cloned.FinalSubmissionDraft == nil || cloned.FinalSubmissionDraft == pkg.FinalSubmissionDraft {
		t.Fatal("final-draft canonical field is missing or shares source state")
	}
	got := cloned.SaleAttributeResolution
	if got == nil || len(got.SKCValueAssignments) != 1 || len(got.SKUValueAssignments) != 1 {
		t.Fatal("public resolution values lost")
	}
	cloned.DraftPayload.SpuName = "mutated-clone"
	cloned.PreviewPayload.SPUName = "mutated-preview"
	cloned.Metadata["owner"] = "clone"
	clonedID := got.SKCValueAssignments["blue"].AttributeValueID
	*clonedID = 999
	if pkg.DraftPayload.SpuName != "original-draft" || pkg.PreviewPayload.SPUName != "original-preview" || pkg.Metadata["owner"] != "source" || *pkg.SaleAttributeResolution.SKCValueAssignments["blue"].AttributeValueID != valueID {
		t.Fatal("clone mutation leaked into source package")
	}
}
func TestPublicResolutionMapsDoNotSharePointersOrSlices(t *testing.T) {
	id := 701
	p := &Package{SaleAttributeResolution: &SaleAttributeResolution{
		SKCValueAssignments: map[string]ResolvedSaleAttribute{"blue": {AttributeValueID: &id}},
		SKUValueAssignments: map[string]ResolvedSaleAttribute{"xl": {AttributeValueID: &id}},
		SKUAttributes:       []ResolvedSaleAttribute{{AttributeValueID: &id}},
	}}
	c, err := ClonePackageForPersistence(p)
	if err != nil {
		t.Fatal(err)
	}
	*c.SaleAttributeResolution.SKCValueAssignments["blue"].AttributeValueID = 999
	*c.SaleAttributeResolution.SKUValueAssignments["xl"].AttributeValueID = 999
	*c.SaleAttributeResolution.SKUAttributes[0].AttributeValueID = 999
	if *p.SaleAttributeResolution.SKCValueAssignments["blue"].AttributeValueID != 701 || *p.SaleAttributeResolution.SKUValueAssignments["xl"].AttributeValueID != 701 || *p.SaleAttributeResolution.SKUAttributes[0].AttributeValueID != 701 {
		t.Fatal("public assignment clone leaked")
	}
}

func TestSaleAttributeResolutionPrivateCloneSchemaIsExplicit(t *testing.T) {
	typeOf := reflect.TypeOf(SaleAttributeResolution{})
	var privateFields []string
	for i := 0; i < typeOf.NumField(); i++ {
		field := typeOf.Field(i)
		if field.PkgPath != "" {
			privateFields = append(privateFields, field.Name)
		}
	}
	sort.Strings(privateFields)
	var want []string
	if !reflect.DeepEqual(privateFields, want) {
		t.Fatalf("private clone schema changed: got %v, want %v; update ClonePackageForPersistence explicitly", privateFields, want)
	}
}

func TestCloneProductForSubmitDeepCopiesProduct(t *testing.T) {
	t.Parallel()

	product := &previewdata.Product{
		SupplierCode: "SUP-1",
		ImageInfo: &previewdata.ImageInfo{
			ImageInfoList: []previewdata.ImageDetail{{ImageURL: "https://cdn.example.com/1.jpg"}},
		},
	}

	cloned, err := CloneProductForSubmit(product)
	if err != nil {
		t.Fatalf("CloneProductForSubmit() error = %v", err)
	}
	if cloned == nil || cloned.SupplierCode != product.SupplierCode {
		t.Fatalf("cloned product = %+v, want supplier code copied", cloned)
	}
	cloned.ImageInfo.ImageInfoList[0].ImageURL = "https://cdn.example.com/changed.jpg"
	if product.ImageInfo.ImageInfoList[0].ImageURL == cloned.ImageInfo.ImageInfoList[0].ImageURL {
		t.Fatal("CloneProductForSubmit() did not deep copy image list")
	}
}

func TestClonePackageImageSetDeepCopiesSlices(t *testing.T) {
	t.Parallel()

	src := &ImageSet{
		MainImage:    "https://cdn.example.com/main.jpg",
		WhiteBgImage: "https://cdn.example.com/white.jpg",
		Gallery:      []string{"https://cdn.example.com/1.jpg"},
		SourceImages: []string{"https://cdn.example.com/source.jpg"},
	}

	pkg, err := ClonePackageForPersistence(&Package{Images: src})
	if err != nil {
		t.Fatal(err)
	}
	cloned := pkg.Images
	if cloned == nil {
		t.Fatal("ClonePackageForPersistence().Images = nil, want clone")
	}
	src.Gallery[0] = "https://cdn.example.com/changed-gallery.jpg"
	src.SourceImages[0] = "https://cdn.example.com/changed-source.jpg"

	if cloned.MainImage != "https://cdn.example.com/main.jpg" || cloned.WhiteBgImage != "https://cdn.example.com/white.jpg" {
		t.Fatalf("clone scalar fields = %+v, want original values", cloned)
	}
	if cloned.Gallery[0] != "https://cdn.example.com/1.jpg" {
		t.Fatalf("clone gallery = %+v, want deep copy", cloned.Gallery)
	}
	if cloned.SourceImages[0] != "https://cdn.example.com/source.jpg" {
		t.Fatalf("clone source images = %+v, want deep copy", cloned.SourceImages)
	}
}
