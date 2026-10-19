package supplychainapp

import (
	"context"
	"sort"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	marketplace "task-processor/internal/marketplace/walmart"
	"time"
)

func (a *Application) PublicationStatus(ctx context.Context, sourceID, storeID string, receiptIDs ...string) (PublicationView, error) {
	view, e := a.Publication(ctx, sourceID, storeID)
	if e != nil {
		return view, e
	}
	if view.Product != nil {
		if a.SheinPublication == nil {
			return view, record.ErrNotReady
		}
		if len(receiptIDs) > 1 || len(receiptIDs) == 1 && receiptIDs[0] != "" && receiptIDs[0] != view.ReceiptID {
			return view, record.ErrForbidden
		}
		status, err := a.SheinPublication.Status(ctx, sourceID, storeID, view.ReceiptID, 0)
		if err != nil {
			return view, err
		}
		view.SheinStatus = &status
		return view, nil
	}
	if view.Walmart == nil {
		return a.TikTokPublicationStatus(ctx, sourceID, storeID)
	}
	id := ""
	if len(receiptIDs) > 0 {
		id = receiptIDs[0]
	}
	if len(receiptIDs) > 1 {
		return view, record.ErrInvalid
	}
	return a.walmartPublicationStatus(ctx, view, id)
}
func (a *Application) walmartPublicationStatus(ctx context.Context, view PublicationView, requested string) (PublicationView, error) {
	if view.Walmart == nil || a.WalmartStatusStores == nil || a.WalmartResults == nil {
		return view, record.ErrNotReady
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	scope, e := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if e != nil {
		return view, e
	}
	selected, e := a.Sources.Select(ctx, view.SourceID)
	if e != nil {
		return view, e
	}
	owner, source, _, e := selected.Read(ctx)
	if e != nil || owner != scope {
		return view, record.ErrForbidden
	}
	binding, e := a.TargetMerchants.ReadTargetMerchant(ctx, scope, view.StoreID)
	if e != nil || binding != view.binding || !binding.ValidWalmartApplication() {
		return view, record.ErrForbidden
	}
	receiptID := view.ReceiptID
	if requested != "" && requested != receiptID {
		found := false
		for _, result := range view.WalmartResults {
			if result.ReceiptID == requested {
				found = true
				break
			}
		}
		if !found {
			return view, record.ErrForbidden
		}
		receiptID = requested
	}
	req, e := a.WalmartResults.BeginWalmartRead(ctx, scope, receiptID)
	if e != nil {
		return view, e
	}
	if req.Receipt.ProductKey != source.Source.ProductKey || req.Receipt.Target.StoreID != view.StoreID || !submission.WalmartCompatibleBinding(req.Receipt.Binding, binding) {
		return view, record.ErrConflict
	}
	observation := marketplace.FeedObservation{FeedID: req.Receipt.Walmart.FeedID, Partial: true, ObservedAt: time.Now().UTC()}
	expected := submission.WalmartExpectedSKUs(req.Receipt.Walmart.SKUs)
	recheck := func(ctx context.Context) error {
		p, e := a.Sources.Select(ctx, view.SourceID)
		if e != nil {
			return e
		}
		s, v, _, e := p.Read(ctx)
		if e != nil || s != scope || v != source {
			return record.ErrForbidden
		}
		b, e := a.TargetMerchants.ReadTargetMerchant(ctx, scope, view.StoreID)
		if e != nil || b != binding {
			return record.ErrForbidden
		}
		return nil
	}
	remoteCtx, remoteCancel := context.WithTimeout(ctx, 10*time.Second)
	defer remoteCancel()
	for offset := 0; offset < len(expected); offset += 50 {
		if e = recheck(ctx); e != nil {
			return view, e
		}
		reader, e := a.WalmartStatusStores.WalmartMerchant(ctx, scope, view.StoreID, &binding)
		if e != nil {
			return view, e
		}
		page, e := reader.FeedPage(remoteCtx, req.Receipt.Walmart.FeedID, offset)
		if e != nil {
			break
		}
		if offset == 0 {
			observation = page
		} else {
			if page.FeedStatus != observation.FeedStatus || page.Received != observation.Received || page.Succeeded != observation.Succeeded || page.Failed != observation.Failed || page.Processing != observation.Processing {
				observation.Partial = true
				break
			}
			observation.Items = append(observation.Items, page.Items...)
			observation.ResponseHashes = append(observation.ResponseHashes, page.ResponseHashes...)
		}
		if page.FeedStatus == "ERROR" {
			break
		}
		if len(page.Items) < 50 {
			break
		}
	}
	observation.Partial = !observation.CompleteFor(expected)
	observation.ObservedAt = time.Now().UTC()
	// Publication/lifecycle are read independently. Absence/failure remains
	// unknown even for a successful ingestion and never grants another send.

	priorProducts := map[string]marketplace.ItemResult{}
	for _, prior := range view.WalmartResults {
		if prior.ReceiptID != req.Receipt.ID {
			continue
		}
		for _, item := range prior.Observation.Items {
			if _, ok := priorProducts[item.SKU]; !ok && item.PublicationObservedAt != nil {
				priorProducts[item.SKU] = item
			}
		}
	}
	tracked := map[string]bool{}
	for _, sku := range req.TrackedSKUs {
		tracked[sku] = true
	}
	indexes := []int{}
	for i, item := range observation.Items {
		if p, ok := priorProducts[item.SKU]; ok {
			observation.Items[i].PublicationStatus = p.PublicationStatus
			observation.Items[i].LifecycleStatus = p.LifecycleStatus
			observation.Items[i].Availability = p.Availability
			observation.Items[i].PublicationObservedAt = p.PublicationObservedAt
		}
		if item.IngestionStatus == "SUCCESS" && tracked[item.SKU] {
			indexes = append(indexes, i)
		}
	}
	sort.SliceStable(indexes, func(i, j int) bool {
		a, b := observation.Items[indexes[i]].PublicationObservedAt, observation.Items[indexes[j]].PublicationObservedAt
		return a == nil && b != nil || a != nil && b != nil && a.Before(*b)
	})
	for n, i := range indexes {
		if n >= 10 {
			break
		}
		reader, e := a.WalmartStatusStores.WalmartMerchant(remoteCtx, scope, view.StoreID, &binding)
		if e != nil {
			break
		}
		product, e := reader.ProductBySKU(remoteCtx, observation.Items[i].SKU)
		if e != nil {
			break
		}
		if product != nil {
			now := time.Now().UTC()
			observation.Items[i].PublicationStatus = product.PublicationStatus
			observation.Items[i].LifecycleStatus = product.LifecycleStatus
			observation.Items[i].Availability = product.Availability
			observation.Items[i].PublicationObservedAt = &now
		}
	}

	if e = recheck(ctx); e != nil {
		return view, e
	}
	if _, e = a.WalmartResults.SaveWalmartRead(ctx, scope, req, observation); e != nil {
		return view, e
	}
	view.WalmartResults, e = a.WalmartResults.ReadWalmartResults(ctx, scope, req.Receipt.Target)
	return view, e
}
