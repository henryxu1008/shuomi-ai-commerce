package productsourcing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

type PlanAcquisition interface {
	Acquire(context.Context, string, string) (sourcing.AcquisitionResult, error)
	Verify(context.Context, string, string) (sourcing.AcquisitionResult, error)
	Read(context.Context, string) (sourcing.AcquisitionResult, error)
}
type PlanCollections interface {
	Mutate(context.Context, string, collection.Mutation) (collection.Receipt, error)
	ReadItem(context.Context, string) (collection.ItemDetail, error)
	ReadBatch(context.Context, string) (collection.Batch, error)
}
type PlanPreparations interface {
	Transfer(context.Context, string, preparation.TransferInput) (preparation.TransferReceipt, error)
	ReadByKey(context.Context, string) (preparation.TransferReceipt, error)
}
type AcquisitionPlanService struct {
	repository   sourcing.AcquisitionPlanRepository
	rules        sourcing.AcquisitionRuleRepository
	child        PlanAcquisition
	collections  PlanCollections
	preparations PlanPreparations
	auth         sourcing.PlanAuthorizer
	discover     sourcing.PublicProductDiscoverer
}

func NewAcquisitionPlanService(repo sourcing.AcquisitionPlanRepository, rules sourcing.AcquisitionRuleRepository, child PlanAcquisition, collections PlanCollections, preps PlanPreparations, auth sourcing.PlanAuthorizer, discover sourcing.PublicProductDiscoverer) (*AcquisitionPlanService, error) {
	if repo == nil || child == nil || auth == nil {
		return nil, sourcing.ErrAcquisitionUnavailable
	}
	return &AcquisitionPlanService{repo, rules, child, collections, preps, auth, discover}, nil
}

type CreatePlanInput struct {
	Mode             string   `json:"mode"`
	Inputs           []string `json:"inputs"`
	TemplateID       string   `json:"templateId,omitempty"`
	TemplateRevision int64    `json:"templateRevision,omitempty"`
	AutoTransfer     bool     `json:"autoTransfer"`
}

func (s *AcquisitionPlanService) check(scope sourcing.AcquisitionPlanScope, transfer bool) sourcing.PlanAuthorityCheck {
	return func(ctx context.Context) error {
		current, err := s.auth.AuthorizePlan(ctx, transfer)
		if err != nil {
			return err
		}
		if current != scope {
			return sourcing.ErrPublicationForbidden
		}
		return nil
	}
}
func (s *AcquisitionPlanService) Create(ctx context.Context, key string, input CreatePlanInput) (sourcing.AcquisitionPlan, error) {
	scope, err := s.auth.AuthorizePlan(ctx, input.AutoTransfer)
	if err != nil {
		return sourcing.AcquisitionPlan{}, err
	}
	if !canonicalAcquisitionKey(key) || (input.TemplateID == "") != (input.TemplateRevision == 0) || input.TemplateRevision < 0 {
		return sourcing.AcquisitionPlan{}, sourcing.ErrInvalidAcquisition
	}
	canonical, err := sourcing.NewAcquisitionPlanInput(input.Mode, input.Inputs, sourcing.AcquisitionRule{}, input.AutoTransfer)
	if err != nil {
		return sourcing.AcquisitionPlan{}, err
	}
	existing, readErr := s.repository.ByKey(ctx, scope, key)
	if readErr == nil {
		if existing.Scope != scope || existing.Key != key || existing.Input.Mode != canonical.Mode || !reflect.DeepEqual(existing.Input.Inputs, canonical.Inputs) || existing.Input.AutoTransfer != input.AutoTransfer || existing.Input.TemplateID != input.TemplateID || existing.Input.TemplateRevision != input.TemplateRevision {
			return sourcing.AcquisitionPlan{}, sourcing.ErrAcquisitionConflict
		}
		return existing, nil
	}
	if !errors.Is(readErr, sourcing.ErrAcquisitionNotFound) {
		return sourcing.AcquisitionPlan{}, readErr
	}
	if input.AutoTransfer && (s.collections == nil || s.preparations == nil) {
		return sourcing.AcquisitionPlan{}, sourcing.ErrAcquisitionUnavailable
	}
	rule := sourcing.AcquisitionRule{}
	if input.TemplateID != "" {
		if s.rules == nil || !canonicalAcquisitionKey(input.TemplateID) {
			return sourcing.AcquisitionPlan{}, sourcing.ErrInvalidAcquisition
		}
		template, e := s.rules.ReadRule(ctx, scope, input.TemplateID)
		if e != nil {
			return sourcing.AcquisitionPlan{}, e
		}
		if template.Archived || template.Revision != input.TemplateRevision {
			return sourcing.AcquisitionPlan{}, sourcing.ErrAcquisitionConflict
		}
		rule = template.Rule
	}
	effective, err := sourcing.NewAcquisitionPlanInput(input.Mode, input.Inputs, rule, input.AutoTransfer)
	if err != nil {
		return sourcing.AcquisitionPlan{}, err
	}
	effective.TemplateID = input.TemplateID
	effective.TemplateRevision = input.TemplateRevision
	p := sourcing.AcquisitionPlan{ID: collection.StableID(scope.OrganizationID, scope.ActorID, "acquisition-plan", key), Key: key, Scope: scope, Input: effective, Revision: 1, Items: []sourcing.AcquisitionPlanItem{}, DiscoveryFailures: []sourcing.DiscoveryFailure{}, State: "pending", CreatedAt: time.Now().UTC()}
	if input.Mode == sourcing.DiscoveryProduct {
		p.DiscoveryIndex = len(effective.Inputs)
		for _, source := range effective.Inputs {
			p.Items = append(p.Items, sourcing.AcquisitionPlanItem{Source: source, ChildKey: planChildKey(p.ID, source), State: "pending"})
		}
	}
	return s.repository.Create(ctx, p, s.check(scope, input.AutoTransfer))
}
func (s *AcquisitionPlanService) Read(ctx context.Context, id string) (sourcing.AcquisitionPlan, error) {
	scope, err := s.auth.AuthorizePlan(ctx, false)
	if err != nil {
		return sourcing.AcquisitionPlan{}, err
	}
	if !canonicalAcquisitionKey(id) {
		return sourcing.AcquisitionPlan{}, sourcing.ErrInvalidAcquisition
	}
	return s.repository.Read(ctx, scope, id)
}
func (s *AcquisitionPlanService) List(ctx context.Context, after string) ([]sourcing.AcquisitionPlan, error) {
	scope, err := s.auth.AuthorizePlan(ctx, false)
	if err != nil {
		return nil, err
	}
	if after != "" && !canonicalAcquisitionKey(after) {
		return nil, sourcing.ErrInvalidAcquisition
	}
	return s.repository.List(ctx, scope, after)
}
func (s *AcquisitionPlanService) ByKey(ctx context.Context, key string) (sourcing.AcquisitionPlan, error) {
	scope, err := s.auth.AuthorizePlan(ctx, false)
	if err != nil {
		return sourcing.AcquisitionPlan{}, err
	}
	if !canonicalAcquisitionKey(key) {
		return sourcing.AcquisitionPlan{}, sourcing.ErrInvalidAcquisition
	}
	return s.repository.ByKey(ctx, scope, key)
}

func (s *AcquisitionPlanService) Advance(ctx context.Context, id string, expected int64) (sourcing.AcquisitionPlan, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	p, err := s.Read(ctx, id)
	if err != nil {
		return p, err
	}
	if p.State == "completed" {
		return p, nil
	}
	check := s.check(p.Scope, p.Input.AutoTransfer)
	if err = check(ctx); err != nil {
		return p, err
	}
	p, err = s.repository.Claim(ctx, p.Scope, id, expected, check)
	if err != nil {
		return p, err
	}
	p.FailureCode = ""
	p.Items = append([]sourcing.AcquisitionPlanItem{}, p.Items...)
	if p.DiscoveryIndex < len(p.Input.Inputs) {
		if s.discover == nil {
			p.State = "paused"
			p.FailureCode = "discovery_unavailable"
			return s.repository.Save(ctx, p, check)
		}
		discovery, e := sourcing.Canonical1688Discovery(p.Input.Mode, p.Input.Inputs[p.DiscoveryIndex])
		if e != nil {
			return p, e
		}
		evidence, e := s.discover.Discover(ctx, discovery)
		if e != nil {
			p.State = "paused"
			p.FailureCode = "discovery_unavailable"
			if errors.Is(e, sourcing.ErrAcquisitionCapacity) {
				p.FailureCode = "collector_capacity_or_cooldown"
			}
			if errors.Is(e, sourcing.ErrAcquisitionFailed) {
				p.DiscoveryFailures = append(p.DiscoveryFailures, sourcing.DiscoveryFailure{InputIndex: p.DiscoveryIndex, Code: "source_unavailable"})
				p.DiscoveryIndex++
				p.State = "pending"
				p.FailureCode = ""
			}
			return s.repository.Save(ctx, p, check)
		}
		if evidence.SourceURL != discovery.URL || len(evidence.Products) > sourcing.MaxPlanProducts {
			return p, sourcing.ErrInvalidAcquisition
		}
		seen := map[string]bool{}
		for _, item := range p.Items {
			seen[item.Source] = true
		}
		for _, raw := range evidence.Products {
			source, e := sourcing.Canonical1688Source(raw)
			if e != nil || source.URL != raw {
				return p, sourcing.ErrInvalidAcquisition
			}
			if seen[raw] {
				continue
			}
			if len(p.Items) >= sourcing.MaxPlanProducts {
				p.Truncated = true
				break
			}
			seen[raw] = true
			p.Items = append(p.Items, sourcing.AcquisitionPlanItem{Source: raw, ChildKey: planChildKey(p.ID, raw), State: "pending"})
		}
		p.Truncated = p.Truncated || evidence.Truncated
		p.DiscoveryIndex++
		p.State = "pending"
		return s.repository.Save(ctx, p, check)
	}
	for index := range p.Items {
		item := &p.Items[index]
		if item.State == "published" || item.State == "failed" {
			continue
		}
		item.ChildKey = planChildKey(p.ID, item.Source)
		var result sourcing.AcquisitionResult
		if item.State == "needs_verification" {
			result, err = s.child.Verify(ctx, item.ChildKey, item.Source)
		} else {
			result, err = s.child.Acquire(ctx, item.ChildKey, item.Source)
		}
		if err != nil {
			switch {
			case errors.Is(err, orgresource.ErrInsufficientBalance), errors.Is(err, sourcing.ErrAcquisitionCapacity):
				p.State = "paused"
				item.FailureCode = "capacity_or_quota_unavailable"
			case errors.Is(err, sourcing.ErrAcquisitionFailed), errors.Is(err, sourcing.ErrInvalidAcquisition):
				item.State = "failed"
				item.FailureCode = "source_unavailable"
				p.State = "pending"
			default:
				item.State = "needs_verification"
				item.FailureCode = "outcome_unknown"
				p.State = "paused"
			}
			return s.repository.Save(ctx, p, check)
		}
		if result.Operation.State == sourcing.AcquisitionFailed {
			item.OperationID = result.Operation.ID
			item.State = "failed"
			item.FailureCode = "source_unavailable"
			p.State = "pending"
			return s.finishOrSave(ctx, p, check)
		}
		if result.Operation.State != sourcing.AcquisitionPublished || result.Operation.Command == nil || result.Publication == nil {
			item.State = "needs_verification"
			p.State = "paused"
			return s.repository.Save(ctx, p, check)
		}
		source, e := sourcing.Canonical1688Source(item.Source)
		if e != nil {
			return p, e
		}
		expectedOperation := acquisitionOperation(sourcing.PublicationScope{OrganizationID: p.Scope.OrganizationID, ActorID: p.Scope.ActorID}, item.ChildKey, source)
		sealed, sealErr := json.Marshal(result.Operation.Command)
		sealedHash := sha256.Sum256(sealed)
		if sealErr != nil || hex.EncodeToString(sealedHash[:]) != result.Operation.CommandHash {
			return p, sourcing.ErrAcquisitionUnavailable
		}
		reference := result.Publication.Receipt
		if result.Operation.ID != expectedOperation.ID || result.Operation.Scope != expectedOperation.Scope || result.Operation.Key != item.ChildKey || result.Operation.Source != source || reference.OrganizationID != p.Scope.OrganizationID || reference.ActorID != p.Scope.ActorID || reference.PublicationID != result.Operation.Command.PublicationID || reference.ProductKey != result.Operation.Command.ProductKey || reference.CatalogVersion == 0 {
			return p, sourcing.ErrAcquisitionUnavailable
		}
		item.OperationID = result.Operation.ID
		facts := []sourcing.AcquisitionPrice{}
		for i := 0; i < 64; i++ {
			raw, ok := result.Operation.Command.Envelope.SupplierOrCostFacts.Facts[fmt.Sprintf("price_fact.%d", i)]
			if !ok {
				break
			}
			var fact sourcing.AcquisitionPrice
			if json.Unmarshal([]byte(raw), &fact) != nil {
				return p, sourcing.ErrAcquisitionUnavailable
			}
			facts = append(facts, fact)
		}
		evaluation, e := p.Input.Rule.Evaluate(facts)
		if e != nil {
			return p, e
		}
		item.Evaluation = &evaluation
		item.State = "published"
		item.FailureCode = ""
		p.State = "pending"
		// Only selected rows need Collection read/manage and Preparation authority.
		if p.Input.AutoTransfer && evaluation.State == sourcing.RuleMatched {
			if s.collections == nil || s.preparations == nil {
				return p, sourcing.ErrAcquisitionUnavailable
			}
			receipt, e := s.collections.Mutate(ctx, collection.StableID(p.ID, item.Source, "collection"), collection.Mutation{Action: "add_acquisition", SourceOperationID: item.OperationID})
			if e != nil {
				return p, e
			}
			detail, e := s.collections.ReadItem(ctx, receipt.ItemID)
			if e != nil {
				return p, e
			}
			if detail.Item.Source.OperationID != item.OperationID || detail.Item.Source.PublicationID != result.Publication.Receipt.PublicationID || detail.Item.Source.Version != result.Publication.Receipt.CatalogVersion || detail.Item.ArchivedAt != nil {
				return p, sourcing.ErrAcquisitionConflict
			}
			item.CollectionItemID = detail.Item.ID
			item.CollectionBatchID = detail.Item.BatchID
			item.CollectionRevision = detail.Item.Revision
		}
		return s.finishOrSave(ctx, p, check)
	}
	return s.finishOrSave(ctx, p, check)
}

func (s *AcquisitionPlanService) finishOrSave(ctx context.Context, p sourcing.AcquisitionPlan, check sourcing.PlanAuthorityCheck) (sourcing.AcquisitionPlan, error) {
	for _, item := range p.Items {
		if item.State != "published" && item.State != "failed" {
			return s.repository.Save(ctx, p, check)
		}
	}
	selected := []string{}
	batchID := ""
	for _, item := range p.Items {
		if item.Evaluation == nil || item.Evaluation.State != sourcing.RuleMatched || !p.Input.AutoTransfer {
			continue
		}
		if item.CollectionItemID == "" || batchID != "" && batchID != item.CollectionBatchID {
			return p, sourcing.ErrAcquisitionConflict
		}
		batchID = item.CollectionBatchID
		selected = append(selected, item.CollectionItemID)
	}
	if len(selected) == 0 {
		p.State = "completed"
		return s.repository.Save(ctx, p, check)
	}
	if len(p.TransferInputJSON) == 0 {
		for _, item := range p.Items {
			if item.CollectionItemID == "" || item.Evaluation == nil || item.Evaluation.State != sourcing.RuleMatched {
				continue
			}
			detail, err := s.collections.ReadItem(ctx, item.CollectionItemID)
			if err != nil {
				return p, err
			}
			if detail.Item.ID != item.CollectionItemID || detail.Item.BatchID != batchID || detail.Item.Revision != item.CollectionRevision || detail.Item.Source.OperationID != item.OperationID || detail.Item.ArchivedAt != nil {
				return p, sourcing.ErrAcquisitionConflict
			}
		}
		batch, err := s.collections.ReadBatch(ctx, batchID)
		if err != nil {
			return p, err
		}
		input := preparation.TransferInput{BatchID: batchID, ExpectedRevision: batch.Revision, ItemIDs: selected}
		p.TransferInputJSON, _ = json.Marshal(input)
		p.TransferKey = collection.StableID(p.ID, "transfer")
		p.State = "transfer_pending"
		// Freeze intent first. The next explicit advance consumes the original key.
		return s.repository.Save(ctx, p, check)
	}
	var input preparation.TransferInput
	if json.Unmarshal(p.TransferInputJSON, &input) != nil || len(input.ItemIDs) == 0 || input.Validate() != nil {
		return p, sourcing.ErrInvalidAcquisition
	}
	receipt, err := s.preparations.ReadByKey(ctx, p.TransferKey)
	if errors.Is(err, preparation.ErrNotFound) {
		receipt, err = s.preparations.Transfer(ctx, p.TransferKey, input)
	}
	if err != nil {
		p.State = "transfer_pending"
		p.FailureCode = "transfer_needs_verification"
		return s.repository.Save(ctx, p, check)
	}
	scope := preparation.Scope{OrganizationID: p.Scope.OrganizationID, ActorID: p.Scope.ActorID, MemberID: p.Scope.MemberID}
	if receipt.Preparation.ID != preparation.OperationID(scope, p.TransferKey) || receipt.Preparation.SourceBatchID != input.BatchID || receipt.Preparation.SourceRevision != input.ExpectedRevision || receipt.Preparation.Count != int64(len(input.ItemIDs)) {
		return p, sourcing.ErrAcquisitionUnavailable
	}
	p.PreparationID = receipt.Preparation.ID
	p.State = "completed"
	p.FailureCode = ""
	return s.repository.Save(ctx, p, check)
}
func planChildKey(plan, source string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("acquisition-plan:"+plan+":"+source)).String()
}

func (s *AcquisitionPlanService) Rules(ctx context.Context) ([]sourcing.AcquisitionRuleTemplate, error) {
	scope, err := s.auth.AuthorizePlan(ctx, false)
	if err != nil {
		return nil, err
	}
	if s.rules == nil {
		return nil, sourcing.ErrAcquisitionUnavailable
	}
	return s.rules.ListRules(ctx, scope)
}
func (s *AcquisitionPlanService) MutateRule(ctx context.Context, key string, input sourcing.RuleTemplateMutation) (sourcing.AcquisitionRuleTemplate, error) {
	scope, err := s.auth.AuthorizePlan(ctx, false)
	if err != nil {
		return sourcing.AcquisitionRuleTemplate{}, err
	}
	if s.rules == nil || !canonicalAcquisitionKey(key) || input.Rule.Validate() != nil {
		return sourcing.AcquisitionRuleTemplate{}, sourcing.ErrInvalidAcquisition
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Action != "archive" && (len(input.Name) == 0 || len(input.Name) > 400 || len([]rune(input.Name)) > 100 || !utf8.ValidString(input.Name) || strings.ContainsAny(input.Name, "\r\n\x00")) {
		return sourcing.AcquisitionRuleTemplate{}, sourcing.ErrInvalidAcquisition
	}
	if input.Action == "create" {
		if input.ID != "" || input.ExpectedRevision != 0 {
			return sourcing.AcquisitionRuleTemplate{}, sourcing.ErrInvalidAcquisition
		}
	} else if (input.Action != "update" && input.Action != "archive") || !canonicalAcquisitionKey(input.ID) || input.ExpectedRevision < 1 {
		return sourcing.AcquisitionRuleTemplate{}, sourcing.ErrInvalidAcquisition
	}
	if input.Action == "archive" && (input.Name != "" || input.Rule != (sourcing.AcquisitionRule{})) {
		return sourcing.AcquisitionRuleTemplate{}, sourcing.ErrInvalidAcquisition
	}
	return s.rules.MutateRule(ctx, scope, key, input, s.check(scope, false))
}
