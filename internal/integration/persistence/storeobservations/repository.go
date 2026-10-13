package storeobservations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"strings"
	"task-processor/internal/authidentity"
	temuo "task-processor/internal/marketplace/temu/observations"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	"time"
)

type Repository struct{ db *gorm.DB }

const productRowsLateral = ` CROSS JOIN LATERAL (
 SELECT k->>'id' AS row_id,jsonb_build_object('product',jsonb_build_object('id',r.platform_id,'skcs',jsonb_build_array(k))) AS payload,k AS item,CASE WHEN k->>'siteStatus'='1' THEN 'active' WHEN k->>'siteStatus'='0' THEN 'off' ELSE 'unknown' END AS sale_state FROM jsonb_array_elements(r.payload->'product'->'skcs') k WHERE jsonb_exists(r.payload,'product')
 UNION ALL SELECT r.platform_id,r.payload,r.payload->'temuProduct',coalesce(r.payload->'temuProduct'->>'saleState','unknown') WHERE jsonb_exists(r.payload,'temuProduct')
 ) entry`

type syncRow struct {
	OrganizationID, StoreID, ID, CommandID, ActorID, MemberID, CommandKey, Kind, Status, ErrorCode string
	Generation, Revision                                                                           int64
	Binding, Progress, QueryRange                                                                  []byte
	CreatedAt                                                                                      time.Time
	ObservedAt                                                                                     *time.Time
}

func (x syncRow) value() (o.Sync, error) {
	s := o.Sync{ID: x.ID, CommandID: x.CommandID, StoreID: x.StoreID, Owner: o.Scope{OrganizationID: x.OrganizationID, ActorID: x.ActorID, MemberID: x.MemberID}, Key: strings.TrimSpace(x.CommandKey), Kind: o.Kind(x.Kind), Status: x.Status, ErrorCode: x.ErrorCode, Generation: x.Generation, Revision: x.Revision, CreatedAt: x.CreatedAt, ObservedAt: x.ObservedAt}
	if json.Unmarshal(x.Binding, &s.Binding) != nil || json.Unmarshal(x.Progress, &s.Progress) != nil {
		return s, o.ErrUnavailable
	}
	if len(x.QueryRange) > 0 && string(x.QueryRange) != "null" && json.Unmarshal(x.QueryRange, &s.Range) != nil {
		return s, o.ErrUnavailable
	}
	return s, nil
}
func readSync(db *gorm.DB, org, id string, lock bool) (o.Sync, error) {
	sql := `SELECT * FROM store_observations.syncs WHERE organization_id=? AND id=?`
	if lock {
		sql += " FOR UPDATE"
	}
	var x syncRow
	r := db.Raw(sql, org, id).Scan(&x)
	if r.Error != nil {
		return o.Sync{}, o.ErrUnavailable
	}
	if r.RowsAffected == 0 {
		return o.Sync{}, o.ErrNotFound
	}
	return x.value()
}
func (r *Repository) ReadSync(ctx context.Context, org, id string) (o.Sync, error) {
	if !authidentity.IsBoundedIdentifier(org) || !o.ValidID(id) {
		return o.Sync{}, o.ErrInvalid
	}
	return readSync(r.db.WithContext(ctx), org, id, false)
}

type commandRow struct {
	OrganizationID, ID, ActorID, MemberID, CommandKey, PayloadHash string
	Input                                                          []byte
	CreatedAt                                                      time.Time
}

func command(db *gorm.DB, scope o.Scope, key string) (o.Command, error) {
	var x commandRow
	result := db.Raw(`SELECT * FROM store_observations.commands WHERE organization_id=? AND actor_id=? AND member_id=? AND command_key=?`, scope.OrganizationID, scope.ActorID, scope.MemberID, key).Scan(&x)
	if result.Error != nil {
		return o.Command{}, o.ErrUnavailable
	}
	if result.RowsAffected == 0 {
		return o.Command{}, o.ErrNotFound
	}
	c := o.Command{ID: x.ID, Owner: scope, Key: key, Hash: strings.TrimSpace(x.PayloadHash), CreatedAt: x.CreatedAt, Syncs: []o.Sync{}}
	if json.Unmarshal(x.Input, &c.Input) != nil {
		return c, o.ErrUnavailable
	}
	var rows []syncRow
	if db.Raw(`SELECT * FROM store_observations.syncs WHERE organization_id=? AND command_id=? ORDER BY store_id`, scope.OrganizationID, c.ID).Scan(&rows).Error != nil {
		return c, o.ErrUnavailable
	}
	for _, row := range rows {
		s, e := row.value()
		if e != nil {
			return c, e
		}
		c.Syncs = append(c.Syncs, s)
	}
	return c, nil
}
func (r *Repository) CommandByKey(ctx context.Context, scope o.Scope, key string) (o.Command, error) {
	if !scope.Valid() || !o.ValidID(key) {
		return o.Command{}, o.ErrInvalid
	}
	return command(r.db.WithContext(ctx), scope, key)
}
func (r *Repository) Begin(ctx context.Context, scope o.Scope, key, hash string, in o.BeginInput, children []o.Sync) (o.Command, error) {
	if !scope.Valid() || !o.ValidID(key) || !o.Text(hash, 64) || hash == "" || !in.Kind.Valid() || len(children) == 0 || len(children) > 500 || len(children) != len(in.Stores) {
		return o.Command{}, o.ErrInvalid
	}
	var out o.Command
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serializes only this original member's parent operation, including a lost
		// commit response. A replacement membership never inherits its receipt.
		lockJSON, _ := json.Marshal([]string{scope.OrganizationID, scope.ActorID, key})
		lockHash := sha256.Sum256(lockJSON)
		if tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, hex.EncodeToString(lockHash[:])).Error != nil {
			return o.ErrUnavailable
		}
		prior, e := command(tx, scope, key)
		if e == nil {
			if prior.Hash != hash {
				return o.ErrConflict
			}
			out = prior
			return nil
		}
		if !errors.Is(e, o.ErrNotFound) {
			return e
		}
		var used int64
		if tx.Raw(`SELECT count(*) FROM store_observations.commands WHERE organization_id=? AND actor_id=? AND command_key=?`, scope.OrganizationID, scope.ActorID, key).Scan(&used).Error != nil {
			return o.ErrUnavailable
		}
		if used != 0 {
			return o.ErrConflict
		}
		raw, e := json.Marshal(in)
		if e != nil || len(raw) > 131072 {
			return o.ErrInvalid
		}
		id := uuid.NewString()
		at := children[0].CreatedAt
		if tx.Exec(`INSERT INTO store_observations.commands(organization_id,id,actor_id,member_id,command_key,payload_hash,input,created_at) VALUES(?,?,?,?,?,?,?::jsonb,?)`, scope.OrganizationID, id, scope.ActorID, scope.MemberID, key, hash, string(raw), at).Error != nil {
			return o.ErrUnavailable
		}
		seen := map[string]bool{}
		for i, s := range children {
			if !o.ValidID(s.ID) || !o.ValidID(s.StoreID) || s.StoreID != in.Stores[i] || seen[s.StoreID] || s.Kind != in.Kind || s.Owner != scope || (s.Status != "pending" && s.Status != "suspended") || s.Progress.Page != 1 || !o.Text(s.Key, 64) || s.Key == "" {
				return o.ErrInvalid
			}
			seen[s.StoreID] = true
			var generation int64
			if tx.Raw(`INSERT INTO store_observations.heads(organization_id,store_id,kind,sequence) VALUES(?,?,?,1) ON CONFLICT(organization_id,store_id,kind) DO UPDATE SET sequence=store_observations.heads.sequence+1 RETURNING sequence`, scope.OrganizationID, s.StoreID, s.Kind).Scan(&generation).Error != nil {
				return o.ErrUnavailable
			}
			binding, _ := json.Marshal(s.Binding)
			progress, _ := json.Marshal(s.Progress)
			queryRange, _ := json.Marshal(s.Range)
			if tx.Exec(`INSERT INTO store_observations.syncs(organization_id,store_id,id,command_id,actor_id,member_id,command_key,kind,generation,revision,status,binding,progress,query_range,created_at,error_code) VALUES(?,?,?,?,?,?,?,?,?,1,?,?::jsonb,?::jsonb,?::jsonb,?,?)`, scope.OrganizationID, s.StoreID, s.ID, id, scope.ActorID, scope.MemberID, s.Key, s.Kind, generation, s.Status, string(binding), string(progress), string(queryRange), s.CreatedAt, s.ErrorCode).Error != nil {
				return o.ErrUnavailable
			}
		}
		out, e = command(tx, scope, key)
		return e
	})
	return out, err
}
func (r *Repository) Heads(ctx context.Context, org string, kind o.Kind, stores []string) ([]o.Sync, []o.Sync, error) {
	selected, latest := []o.Sync{}, []o.Sync{}
	if !authidentity.IsBoundedIdentifier(org) || !kind.Valid() {
		return nil, nil, o.ErrInvalid
	}
	if len(stores) == 0 {
		return selected, latest, nil
	}
	for _, id := range stores {
		if !o.ValidID(id) {
			return nil, nil, o.ErrInvalid
		}
	}
	db := r.db.WithContext(ctx)
	var rows []syncRow
	if db.Raw(`SELECT DISTINCT ON(store_id) * FROM store_observations.syncs WHERE organization_id=? AND kind=? AND store_id IN ? ORDER BY store_id,generation DESC`, org, kind, stores).Scan(&rows).Error != nil {
		return nil, nil, o.ErrUnavailable
	}
	byStore := map[string]o.Sync{}
	for _, row := range rows {
		s, e := row.value()
		if e != nil {
			return nil, nil, e
		}
		latest = append(latest, s)
		byStore[s.StoreID] = s
	}
	rows = nil
	if db.Raw(`SELECT s.* FROM store_observations.heads h JOIN store_observations.syncs s ON s.organization_id=h.organization_id AND s.store_id=h.store_id AND s.id=h.current_id AND s.kind=h.kind WHERE h.organization_id=? AND h.kind=? AND h.store_id IN ?`, org, kind, stores).Scan(&rows).Error != nil {
		return nil, nil, o.ErrUnavailable
	}
	for _, row := range rows {
		s, e := row.value()
		if e != nil {
			return nil, nil, e
		}
		byStore[s.StoreID] = s
	}
	for _, store := range stores {
		if s, ok := byStore[store]; ok {
			selected = append(selected, s)
		}
	}
	return selected, latest, nil
}
func sameSync(a, b o.Sync) bool {
	return a.ID == b.ID && a.StoreID == b.StoreID && a.Owner == b.Owner && a.Kind == b.Kind && a.Binding == b.Binding && a.Generation == b.Generation && a.Revision == b.Revision
}
func (r *Repository) CommitPage(ctx context.Context, expected o.Sync, next o.Checkpoint, records []o.Record, status string, at time.Time) (o.Sync, error) {
	if status != "running" && status != "completed" && status != "partial" || !expected.Owner.Valid() || !o.ValidID(expected.ID) || at.IsZero() || len(records) > 30 {
		return o.Sync{}, o.ErrInvalid
	}
	var out o.Sync
	e := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, e := readSync(tx, expected.Owner.OrganizationID, expected.ID, true)
		if e != nil {
			return e
		}
		if !sameSync(current, expected) || current.Terminal() {
			return o.ErrConflict
		}
		// Coverage defects cannot be cleared by a later successful page.
		for _, note := range current.Progress.Notes {
			next.Note(note)
		}
		next.Incomplete = next.Incomplete || current.Progress.Incomplete
		seen := map[string]bool{}
		for _, v := range records {
			if v.StoreID != current.StoreID || v.SyncID != current.ID || !o.Identity(v.ID) || seen[v.ID] || at.Before(current.CreatedAt) {
				return o.ErrInvalid
			}
			seen[v.ID] = true
			validProduct := v.Order == nil && v.Product != nil && v.TemuProduct == nil && v.Product.ID == v.ID && current.Binding.Platform != storecenter.PlatformTemu
			if current.Binding.Platform == storecenter.PlatformTemu {
				validProduct = v.Order == nil && v.Product == nil && v.TemuProduct != nil && v.TemuProduct.ID == v.ID && temuo.ValidProduct(*v.TemuProduct) && v.TemuProduct.Mode == string(current.Binding.MerchantMode)
			}
			if current.Kind == o.Products && !validProduct || current.Kind == o.Orders && (v.Order == nil || v.Product != nil || v.TemuProduct != nil || v.Order.ID != v.ID || v.Order.Site != "shein-us" || current.Binding.Platform == storecenter.PlatformTemu) {
				return o.ErrInvalid
			}
			var prior struct {
				WindowKey string
				PageIndex int
			}
			result := tx.Raw(`SELECT window_key,page_index FROM store_observations.records WHERE organization_id=? AND store_id=? AND sync_id=? AND platform_id=?`, current.Owner.OrganizationID, current.StoreID, current.ID, v.ID).Scan(&prior)
			if result.Error != nil {
				return o.ErrUnavailable
			}
			if result.RowsAffected > 0 && prior.WindowKey == v.WindowKey && prior.PageIndex != current.Progress.Page {
				next.Note("duplicate_platform_identity")
			}
			payload, _ := json.Marshal(struct {
				Product     *o.Product     `json:"product,omitempty"`
				TemuProduct *temuo.Product `json:"temuProduct,omitempty"`
				Order       *o.Order       `json:"order,omitempty"`
			}{v.Product, v.TemuProduct, v.Order})
			if len(payload) > 262144 {
				return o.ErrInvalid
			}
			if tx.Exec(`INSERT INTO store_observations.records(organization_id,store_id,sync_id,kind,platform_id,payload,observed_at,window_key,page_index) VALUES(?,?,?,?,?,?::jsonb,?,?,?) ON CONFLICT(organization_id,store_id,sync_id,platform_id) DO UPDATE SET payload=EXCLUDED.payload,observed_at=EXCLUDED.observed_at,window_key=EXCLUDED.window_key,page_index=EXCLUDED.page_index`, current.Owner.OrganizationID, current.StoreID, current.ID, current.Kind, v.ID, string(payload), at, v.WindowKey, current.Progress.Page).Error != nil {
				return o.ErrUnavailable
			}
		}
		if status == "completed" && next.Incomplete {
			status = "partial"
		}
		raw, _ := json.Marshal(next)
		if len(raw) > 65536 {
			return o.ErrInvalid
		}
		if tx.Exec(`UPDATE store_observations.syncs SET progress=?::jsonb,status=?,revision=revision+1,observed_at=? WHERE organization_id=? AND id=?`, string(raw), status, at, current.Owner.OrganizationID, current.ID).Error != nil {
			return o.ErrUnavailable
		}
		if status == "completed" {
			// Allocation and promotion share this scoped head lock. An older completion
			// can retain its own results but cannot undo the newer published generation.
			if tx.Exec(`UPDATE store_observations.heads h SET current_id=? WHERE h.organization_id=? AND h.store_id=? AND h.kind=? AND (h.current_id IS NULL OR EXISTS(SELECT 1 FROM store_observations.syncs s WHERE s.organization_id=h.organization_id AND s.store_id=h.store_id AND s.kind=h.kind AND s.id=h.current_id AND s.generation<?))`, current.ID, current.Owner.OrganizationID, current.StoreID, current.Kind, current.Generation).Error != nil {
				return o.ErrUnavailable
			}
		}
		out, e = readSync(tx, current.Owner.OrganizationID, current.ID, false)
		return e
	})
	return out, e
}
func (r *Repository) Stop(ctx context.Context, expected o.Sync, status, code string) (o.Sync, error) {
	if (status != "failed" && status != "suspended") || !o.Text(code, 64) || !expected.Owner.Valid() || !o.ValidID(expected.ID) {
		return o.Sync{}, o.ErrInvalid
	}
	var out o.Sync
	e := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, e := readSync(tx, expected.Owner.OrganizationID, expected.ID, true)
		if e != nil {
			return e
		}
		if current.Terminal() {
			out = current
			return nil
		}
		if !sameSync(current, expected) {
			return o.ErrConflict
		}
		if tx.Exec(`UPDATE store_observations.syncs SET status=?,error_code=?,revision=revision+1 WHERE organization_id=? AND id=?`, status, code, current.Owner.OrganizationID, current.ID).Error != nil {
			return o.ErrUnavailable
		}
		out, e = readSync(tx, current.Owner.OrganizationID, current.ID, false)
		return e
	})
	return out, e
}

type recordRow struct {
	StoreID, SyncID, PlatformID, RowID string
	Payload                            []byte
	ObservedAt                         time.Time
}

func (x recordRow) value() (o.Record, error) {
	v := o.Record{ID: x.PlatformID, StoreID: x.StoreID, SyncID: x.SyncID, ObservedAt: x.ObservedAt}
	if json.Unmarshal(x.Payload, &v) != nil {
		return v, o.ErrUnavailable
	}
	return v, nil
}
func (r *Repository) Record(ctx context.Context, org, store string, kind o.Kind, sync, id string) (o.Record, error) {
	if !authidentity.IsBoundedIdentifier(org) || !o.ValidID(store) || !kind.Valid() || !o.ValidID(sync) || !o.Identity(id) {
		return o.Record{}, o.ErrInvalid
	}
	var row recordRow
	q := r.db.WithContext(ctx).Raw(`SELECT store_id,sync_id,platform_id,payload,observed_at FROM store_observations.records WHERE organization_id=? AND store_id=? AND kind=? AND sync_id=? AND platform_id=?`, org, store, kind, sync, id).Scan(&row)
	if q.Error != nil {
		return o.Record{}, o.ErrUnavailable
	}
	if q.RowsAffected == 0 {
		return o.Record{}, o.ErrNotFound
	}
	return row.value()
}

type pageCursor struct {
	Scope string `json:"scope"`
	Store string `json:"store"`
	ID    string `json:"id"`
	Row   string `json:"row"`
}

func queryHash(org string, q o.Query) string {
	raw, _ := json.Marshal(struct {
		Org                       string
		Kind                      o.Kind
		Keyword, Status, Platform string
		Sources                   map[string]string
	}{org, q.Kind, q.Keyword, q.Status, q.Platform, q.Sources})
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func (r *Repository) List(ctx context.Context, org string, q o.Query) (out o.Result, err error) {
	// Pending generations can receive a page while read. Summary and rows still
	// come from one SQL snapshot; completed generations remain immutable.
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { var e error; out, e = (&Repository{db: tx}).list(ctx, org, q); return e }, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return out, err
}
func (r *Repository) list(ctx context.Context, org string, q o.Query) (o.Result, error) {
	out := o.Result{Items: []o.Record{}}
	if !authidentity.IsBoundedIdentifier(org) || !q.Kind.Valid() || q.Limit < 1 || q.Limit > 50 || !o.Text(q.Keyword, 200) || len(q.After) > 4096 || len(q.Sources) > 500 || q.RecordByteLimit < 0 || q.RecordByteLimit > 1<<20 {
		return out, o.ErrInvalid
	}
	fingerprint := queryHash(org, q)
	var cursor pageCursor
	if q.After != "" {
		raw, e := base64.RawURLEncoding.DecodeString(q.After)
		if e != nil || json.Unmarshal(raw, &cursor) != nil || cursor.Scope != fingerprint || !o.ValidID(cursor.Store) || !o.Identity(cursor.ID) || !o.Text(cursor.Row, 128) {
			return out, o.ErrConflict
		}
	}
	if len(q.Sources) == 0 {
		return out, nil
	}
	conditions, args := []string{}, []any{org, q.Kind}
	for store, sync := range q.Sources {
		if !o.ValidID(store) || !o.ValidID(sync) {
			return out, o.ErrInvalid
		}
		conditions = append(conditions, "(r.store_id=? AND r.sync_id=?)")
		args = append(args, store, sync)
	}
	base := `SELECT r.store_id,r.sync_id,r.platform_id,r.observed_at,`
	if q.Kind == o.Products {
		base += `entry.row_id,entry.payload,entry.item,entry.sale_state FROM store_observations.records r` + productRowsLateral
	} else {
		base += `r.platform_id AS row_id,r.payload,r.payload->'order' AS item FROM store_observations.records r`
	}
	base += ` WHERE r.organization_id=? AND r.kind=? AND (` + strings.Join(conditions, " OR ") + `)`
	filter := " WHERE true"
	if q.Keyword != "" {
		pattern := "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(q.Keyword) + "%"
		if q.Kind == o.Products {
			filter += ` AND (platform_id ILIKE ? OR row_id ILIKE ? OR item->>'title' ILIKE ? OR item->>'sellerCode' ILIKE ? OR item->>'externalCode' ILIKE ? OR item->>'productId' ILIKE ? OR EXISTS(SELECT 1 FROM jsonb_array_elements(item->'skus') sku WHERE sku->>'id' ILIKE ? OR sku->>'sellerSku' ILIKE ? OR sku->>'externalCode' ILIKE ? OR sku->>'serialNumber' ILIKE ?))`
			for i := 0; i < 10; i++ {
				args = append(args, pattern)
			}
		} else {
			filter += ` AND (platform_id ILIKE ? OR EXISTS(SELECT 1 FROM jsonb_array_elements(item->'items') goods WHERE goods->>'title' ILIKE ? OR goods->>'sku' ILIKE ? OR goods->>'sellerSku' ILIKE ?))`
			for i := 0; i < 4; i++ {
				args = append(args, pattern)
			}
		}
	}
	exceptional := `(jsonb_array_length(item->'reasons')>0 OR item->>'tag'='1' OR item->>'status' IN('8','9') OR EXISTS(SELECT 1 FROM jsonb_array_elements(item->'packages') p WHERE p->>'label' IN('1','2')))`
	if q.Status != "" {
		if q.Kind == o.Products {
			switch q.Status {
			case "active":
				filter += ` AND sale_state='active'`
			case "off":
				filter += ` AND sale_state='off'`
			case "unknown":
				filter += ` AND sale_state NOT IN('active','off')`
			default:
				return out, o.ErrInvalid
			}
		} else {
			switch q.Status {
			case "transit":
				filter += ` AND item->>'status' IN('4','7')`
			case "exceptional":
				filter += " AND " + exceptional
			case "unknown":
				filter += ` AND (item->>'status' IS NULL OR item->>'status' NOT IN('1','2','3','4','5','6','7','8','9'))`
			default:
				if !strings.Contains("|1|2|3|4|5|6|7|8|9|", "|"+q.Status+"|") {
					return out, o.ErrInvalid
				}
				filter += ` AND item->>'status'=?`
				args = append(args, q.Status)
			}
		}
	}
	cte := `WITH rows AS (` + base + `), filtered AS (SELECT * FROM rows` + filter + `) `
	summary := `SELECT count(*) AS total`
	summaryArgs := append([]any{}, args...)
	if q.Kind == o.Products {
		summary += `,count(*) FILTER(WHERE sale_state='active') AS active,count(*) FILTER(WHERE sale_state='off') AS off_shelf,count(*) FILTER(WHERE sale_state NOT IN('active','off')) AS unknown`
	} else {
		summary += `,count(*) FILTER(WHERE NULLIF(item->>'createdAt','')::timestamptz>=? AND NULLIF(item->>'createdAt','')::timestamptz<?) AS today,count(*) FILTER(WHERE NULLIF(item->>'createdAt','') IS NULL) AS today_unknown,count(*) FILTER(WHERE item->>'status' IN('1','2','3')) AS pending,count(*) FILTER(WHERE item->>'status' IN('4','7')) AS transit,count(*) FILTER(WHERE ` + exceptional + `) AS exceptional,count(*) FILTER(WHERE item->>'status' IS NULL OR item->>'status' NOT IN('1','2','3','4','5','6','7','8','9')) AS unknown`
		summaryArgs = append(summaryArgs, q.TodayStart, q.TodayStart.Add(24*time.Hour))
	}
	if r.db.WithContext(ctx).Raw(cte+summary+` FROM filtered`, summaryArgs...).Scan(&out.Summary).Error != nil {
		return out, o.ErrUnavailable
	}
	page := cte + `SELECT store_id,sync_id,platform_id,row_id,payload,observed_at FROM filtered`
	if q.After != "" {
		page += ` WHERE (store_id::text,platform_id,row_id)>(?,?,?)`
		args = append(args, cursor.Store, cursor.ID, cursor.Row)
	}
	page += ` ORDER BY store_id,platform_id,row_id LIMIT ?`
	args = append(args, q.Limit+1)
	var rows []recordRow
	if r.db.WithContext(ctx).Raw(page, args...).Scan(&rows).Error != nil {
		return out, o.ErrUnavailable
	}
	// Bound encoded observations as well as row count. A legitimate many-SKU
	// product must remain pageable instead of making every HTTP response too big.
	encodedBytes, more := 2, false
	byteLimit := 1 << 20
	if q.RecordByteLimit > 0 {
		byteLimit = q.RecordByteLimit
	}
	for _, row := range rows {
		if len(out.Items) == q.Limit {
			more = true
			break
		}
		v, e := row.value()
		if e != nil {
			return out, e
		}
		payload, e := json.Marshal(v)
		if e != nil {
			return out, o.ErrUnavailable
		}
		size := encodedBytes + len(payload)
		if len(out.Items) > 0 {
			size++
		}
		if size > byteLimit {
			if len(out.Items) == 0 {
				return out, o.ErrUnavailable
			}
			more = true
			break
		}
		encodedBytes = size
		out.Items = append(out.Items, v)
	}
	if more {
		last := rows[len(out.Items)-1]
		raw, _ := json.Marshal(pageCursor{fingerprint, last.StoreID, last.PlatformID, last.RowID})
		out.Next = base64.RawURLEncoding.EncodeToString(raw)
	}
	return out, nil
}

var _ o.Repository = (*Repository)(nil)

// StoreAggregates projects current source facts through the admitted persistence port.
func (r *Repository) StoreAggregates(ctx context.Context, org string, kind o.Kind, sources map[string]string, inventory bool) (out map[string]o.StoreAggregate, err error) {
	out = map[string]o.StoreAggregate{}
	if !authidentity.IsBoundedIdentifier(org) || !kind.Valid() || len(sources) > 500 {
		return nil, o.ErrInvalid
	}
	if len(sources) == 0 {
		return out, nil
	}
	conditions, args := []string{}, []any{org, kind}
	for store, sync := range sources {
		if !o.ValidID(store) || !o.ValidID(sync) {
			return nil, o.ErrInvalid
		}
		conditions = append(conditions, "(r.store_id=? AND r.sync_id=?)")
		args = append(args, store, sync)
		out[store] = o.StoreAggregate{}
	}
	base := ` FROM store_observations.records r`
	if kind == o.Products {
		base += productRowsLateral + ` CROSS JOIN LATERAL (SELECT entry.item) projection(k)`
	}
	where := ` WHERE r.organization_id=? AND r.kind=? AND (` + strings.Join(conditions, " OR ") + `)`
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := `SELECT r.store_id, count(*) FILTER(WHERE r.payload->'order'->>'status' IN('1','2','3')) AS pending,count(*) FILTER(WHERE r.payload->'order'->>'status' IS NULL OR r.payload->'order'->>'status' NOT IN('1','2','3','4','5','6','7','8','9')) AS status_unknown`
		if kind == o.Products {
			query = `SELECT r.store_id,count(*) FILTER(WHERE entry.sale_state='active') AS active,count(*) FILTER(WHERE entry.sale_state NOT IN('active','off')) AS status_unknown`
		}
		var totals []struct {
			StoreID                        string
			Active, Pending, StatusUnknown int64
		}
		if err := tx.Raw(query+base+where+` GROUP BY r.store_id`, args...).Scan(&totals).Error; err != nil {
			return err
		}
		for _, total := range totals {
			out[total.StoreID] = o.StoreAggregate{Active: total.Active, Pending: total.Pending, StatusUnknown: total.StatusUnknown}
		}
		if kind != o.Products || !inventory {
			return nil
		}
		// Transfer only inventory identities and quantities, never full Product detail.
		// Local TEMU stock is SKU aggregate stock. Its internal counting bucket is
		// not an official physical warehouse and is never persisted or exposed as one.
		rows, err := tx.Raw(`SELECT r.store_id,coalesce(sku->>'id','missing:'||r.platform_id||':'||(k->>'id')),CASE WHEN k->>'entity'='local_goods' THEN 'temu:local:aggregate-stock' ELSE coalesce(stock->>'warehouseId','') END,CASE WHEN jsonb_typeof(stock->'quantity')='number' THEN (stock->>'quantity')::bigint WHEN k->>'entity'='local_goods' AND sku->>'stock' ~ '^[0-9]{1,19}$' THEN (sku->>'stock')::bigint ELSE NULL END`+base+` LEFT JOIN LATERAL jsonb_array_elements(k->'skus') sku ON true LEFT JOIN LATERAL jsonb_array_elements(sku->'inventory') stock ON true`+where, args...).Rows()
		if err != nil {
			return err
		}
		defer rows.Close()
		count := 0
		for rows.Next() {
			count++
			if count > 100000 {
				return o.ErrUnavailable
			}
			var store string
			var fact o.InventoryFact
			if err = rows.Scan(&store, &fact.SKU, &fact.Warehouse, &fact.Quantity); err != nil {
				return err
			}
			value := out[store]
			value.Inventory = append(value.Inventory, fact)
			out[store] = value
		}
		return rows.Err()
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, o.ErrUnavailable
	}
	return out, nil
}
