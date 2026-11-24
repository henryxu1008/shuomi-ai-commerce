package storecenter

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	meta "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/metacatalog"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type MetaCatalogCipher interface {
	KeyID() string
	Seal([]byte, []byte) (string, error)
	Open(string, string, []byte) ([]byte, error)
}

type MetaCatalogSimulation struct {
	repository *MemberScopedStoreRepository
	provider   meta.Provider
	cipher     MetaCatalogCipher
	callback   string
}

func NewMetaCatalogSimulation(repo *MemberScopedStoreRepository, provider meta.Provider, cipher MetaCatalogCipher, callback string) (*MetaCatalogSimulation, error) {
	if repo == nil || provider == nil || cipher == nil {
		return nil, ErrOfficialConnectionUnavailable
	}
	if _, err := provider.ConsentURL(callback, "validate"); err != nil {
		return nil, ErrOfficialConnectionUnavailable
	}
	return &MetaCatalogSimulation{repo, provider, cipher, callback}, nil
}

// InstallMetaCatalogSimulationSchema provisions an isolated Store fixture.
func InstallMetaCatalogSimulationSchema(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`ALTER TABLE public.workbench_stores DROP CONSTRAINT store_platform,
   ADD CONSTRAINT store_platform CHECK (platform IN ('shein','temu','tiktokshop','walmart','amazon','googlemerchant','metacatalog'))`).Error; err != nil {
			return err
		}
		return tx.Exec(`CREATE TABLE public.workbench_store_meta_catalog_simulation (
   organization_id VARCHAR(200) NOT NULL, store_id CHAR(36) NOT NULL,
   attempt_id CHAR(36) NOT NULL, actor_id VARCHAR(200) NOT NULL, member_id VARCHAR(200) NOT NULL,
   store_version BIGINT NOT NULL CHECK (store_version > 0),
   generation BIGINT NOT NULL CHECK (generation > 0), state_hash CHAR(64) NOT NULL,
   phase TEXT NOT NULL CHECK (phase IN ('consent','exchanging','candidate','connected','disconnected')),
   expires_at TIMESTAMPTZ NOT NULL, key_id TEXT NOT NULL, ciphertext TEXT NOT NULL,
   business_id TEXT NOT NULL, catalog_id TEXT UNIQUE, whats_app_id TEXT NOT NULL,
   PRIMARY KEY (organization_id,store_id),
   FOREIGN KEY (organization_id,store_id) REFERENCES public.workbench_stores(organization_id,id),
   CHECK ((phase IN ('candidate','connected')) = (ciphertext <> '' AND key_id <> '')),
   CHECK (phase <> 'connected' OR catalog_id IS NOT NULL)
  )`).Error
	})
}

type metaConnection struct {
	OrganizationID, StoreID, AttemptID, ActorID, MemberID, Phase, KeyID, Ciphertext, BusinessID, WhatsAppID string
	StoreVersion, Generation                                                                                int64
	StateHash                                                                                               string
	ExpiresAt                                                                                               time.Time
	CatalogID                                                                                               *string
}

func (metaConnection) TableName() string { return "workbench_store_meta_catalog_simulation" }

type MetaBinding struct {
	CatalogID  string `json:"catalogId"`
	BusinessID string `json:"businessId"`
	WhatsAppID string `json:"whatsAppId,omitempty"`
	Generation int64  `json:"-"`
}

type MetaConnectionView struct {
	Connected bool  `json:"connected"`
	Version   int64 `json:"version"`
	MetaBinding
}

type MetaChoices struct {
	Catalogs []meta.Choice `json:"catalogs"`
	Accounts []meta.Choice `json:"accounts"`
}

type MetaObservation struct {
	Catalog  string `json:"catalog"`
	WhatsApp string `json:"whatsApp"`
	Facebook string `json:"facebook"`
}

func (r metaConnection) binding() MetaBinding {
	if r.CatalogID == nil {
		return MetaBinding{}
	}
	return MetaBinding{*r.CatalogID, r.BusinessID, r.WhatsAppID, r.Generation}
}

func (s *MetaCatalogSimulation) withStore(ctx context.Context, org, id string, write bool,
	f func(*gorm.DB, StoreMemberAccess, workbenchStoreRecord, metaConnection) error) error {
	if uuid.Validate(id) != nil {
		return ErrNotFound
	}
	access, err := s.repository.authorize(ctx, org, write)
	if err != nil {
		return err
	}
	if write && !access.Administrator {
		return ErrNotFound
	}
	return s.repository.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Transaction(func(tx *gorm.DB) error {
			store, err := connectionStore(tx, access, id)
			if err != nil {
				return err
			}
			if store.Platform != string(PlatformMetaCatalog) || store.RecordStatus != string(RecordStatusActive) {
				return ErrNotFound
			}
			var row metaConnection
			err = tx.Where("organization_id = ? AND store_id = ?", org, id).Take(&row).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			return f(tx, access, store, row)
		})
}

func saveMetaConnection(tx *gorm.DB, row metaConnection) error {
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "organization_id"}, {Name: "store_id"}}, UpdateAll: true}).Create(&row).Error
}

func (s *MetaCatalogSimulation) Read(ctx context.Context, org, id string) (MetaConnectionView, error) {
	var out MetaConnectionView
	err := s.withStore(ctx, org, id, false, func(_ *gorm.DB, _ StoreMemberAccess, store workbenchStoreRecord, row metaConnection) error {
		out = MetaConnectionView{row.connected(store), store.Version, row.binding()}
		return nil
	})
	return out, err
}

func (r metaConnection) connected(store workbenchStoreRecord) bool {
	return r.Phase == "connected" && r.AttemptID == store.ConnectionRef && time.Now().Before(r.ExpiresAt)
}

func metaStateHash(state string) string {
	hash := sha256.Sum256([]byte(state))
	return hex.EncodeToString(hash[:])
}

func (s *MetaCatalogSimulation) Begin(ctx context.Context, org, id string, version int64) (OfficialConnectionBegin, error) {
	var entropy [32]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return OfficialConnectionBegin{}, ErrOfficialConnectionUnavailable
	}
	state := base64.RawURLEncoding.EncodeToString(entropy[:])
	attempt := uuid.NewString()
	consent, err := s.provider.ConsentURL(s.callback, state)
	if err != nil {
		return OfficialConnectionBegin{}, ErrOfficialConnectionUnavailable
	}
	now := time.Now().UTC()
	expires := now.Add(10 * time.Minute)
	err = s.withStore(ctx, org, id, true, func(tx *gorm.DB, a StoreMemberAccess, store workbenchStoreRecord, row metaConnection) error {
		if store.Version != version {
			return ErrVersionConflict
		}
		row.OrganizationID, row.StoreID, row.AttemptID = org, id, attempt
		row.ActorID, row.MemberID, row.StoreVersion = a.ActorID, a.MemberID, store.Version+1
		row.Generation++
		row.Phase, row.StateHash, row.ExpiresAt = "consent", metaStateHash(state), expires
		row.Ciphertext, row.KeyID = "", ""
		if err := saveMetaConnection(tx, row); err != nil {
			return err
		}
		return updateConnectionRef(tx, store, attempt, a.ActorID, now)
	})
	if err != nil {
		return OfficialConnectionBegin{}, err
	}
	return OfficialConnectionBegin{AttemptID: attempt, AuthorizationURL: consent, ExpiresAt: expires}, nil
}

func (s *MetaCatalogSimulation) attempt(ctx context.Context, org, id, attempt, phase string, f func(*gorm.DB, metaConnection) error) error {
	return s.withStore(ctx, org, id, true, func(tx *gorm.DB, a StoreMemberAccess, store workbenchStoreRecord, row metaConnection) error {
		if attempt == "" || row.AttemptID != attempt || store.ConnectionRef != attempt || row.Phase != phase ||
			row.ActorID != a.ActorID || row.MemberID != a.MemberID || row.StoreVersion != store.Version || !time.Now().Before(row.ExpiresAt) {
			return ErrNotFound
		}
		return f(tx, row)
	})
}

func (r metaConnection) aad() []byte {
	data, _ := json.Marshal([]any{"meta-catalog-simulation-v1", r.OrganizationID, r.StoreID, r.AttemptID, r.Phase, r.Generation, r.binding()})
	return data
}

func (s *MetaCatalogSimulation) seal(row *metaConnection, cred meta.Credentials) error {
	data, err := json.Marshal(cred)
	if err != nil {
		return ErrOfficialConnectionUnavailable
	}
	row.Ciphertext, err = s.cipher.Seal(data, row.aad())
	if err != nil {
		return ErrOfficialConnectionUnavailable
	}
	row.KeyID = s.cipher.KeyID()
	return nil
}

func (s *MetaCatalogSimulation) open(row metaConnection) (meta.Credentials, error) {
	var cred meta.Credentials
	data, err := s.cipher.Open(row.KeyID, row.Ciphertext, row.aad())
	if err != nil || json.Unmarshal(data, &cred) != nil || !time.Now().Before(cred.ExpiresAt) {
		return meta.Credentials{}, ErrOfficialConnectionUnavailable
	}
	return cred, nil
}

func (s *MetaCatalogSimulation) Complete(ctx context.Context, org, id, attempt, state, code string) error {
	if len(state) != 43 || code == "" || len(code) > 1024 {
		return ErrNotFound
	}
	err := s.attempt(ctx, org, id, attempt, "consent", func(tx *gorm.DB, row metaConnection) error {
		if subtle.ConstantTimeCompare([]byte(row.StateHash), []byte(metaStateHash(state))) != 1 {
			return ErrNotFound
		}
		row.Phase = "exchanging"
		return saveMetaConnection(tx, row)
	})
	if err != nil {
		return err
	}
	cred, err := s.provider.Exchange(ctx, s.callback, code)
	if err != nil {
		return ErrOfficialConnectionUnavailable
	}
	return s.attempt(ctx, org, id, attempt, "exchanging", func(tx *gorm.DB, row metaConnection) error {
		row.Phase = "candidate"
		if err := s.seal(&row, cred); err != nil {
			return err
		}
		return saveMetaConnection(tx, row)
	})
}

func (s *MetaCatalogSimulation) Choices(ctx context.Context, org, id, attempt string) (MetaChoices, error) {
	var result MetaChoices
	err := s.attempt(ctx, org, id, attempt, "candidate", func(_ *gorm.DB, row metaConnection) error {
		cred, err := s.open(row)
		if err != nil {
			return err
		}
		result.Catalogs, result.Accounts, err = s.provider.Choices(ctx, cred)
		if err != nil {
			return ErrOfficialConnectionUnavailable
		}
		return nil
	})
	return result, err
}

func (s *MetaCatalogSimulation) Bind(ctx context.Context, org, id, attempt, catalogID, accountID string) (MetaConnectionView, error) {
	if !meta.ValidID(catalogID) || accountID != "" && !meta.ValidID(accountID) {
		return MetaConnectionView{}, ErrNotFound
	}
	err := s.attempt(ctx, org, id, attempt, "candidate", func(tx *gorm.DB, row metaConnection) error {
		cred, err := s.open(row)
		if err != nil {
			return err
		}
		catalogs, accounts, err := s.provider.Choices(ctx, cred)
		if err != nil {
			return ErrOfficialConnectionUnavailable
		}
		var business string
		for _, choice := range catalogs {
			if choice.ID == catalogID {
				business = choice.BusinessID
				break
			}
		}
		if business == "" {
			return ErrNotFound
		}
		if row.CatalogID != nil && (*row.CatalogID != catalogID || row.BusinessID != business) {
			return ErrVersionConflict
		}
		if accountID != "" {
			eligible := false
			for _, choice := range accounts {
				if choice.ID == accountID && choice.BusinessID == business {
					eligible = true
					break
				}
			}
			if !eligible {
				return ErrNotFound
			}
			linked, err := s.provider.WhatsAppLinked(ctx, cred.Token, accountID, catalogID)
			if err != nil {
				return ErrOfficialConnectionUnavailable
			}
			if !linked {
				return ErrVersionConflict
			}
		}
		row.CatalogID, row.BusinessID, row.WhatsAppID = &catalogID, business, accountID
		row.Phase, row.ExpiresAt = "connected", cred.ExpiresAt
		if err := s.seal(&row, cred); err != nil {
			return err
		}
		if err := saveMetaConnection(tx, row); err != nil {
			if errors.Is(googleConflict(err), ErrAlreadyExists) {
				return ErrVersionConflict
			}
			return err
		}
		return nil
	})
	if err != nil {
		return MetaConnectionView{}, err
	}
	return s.Read(ctx, org, id)
}

func (s *MetaCatalogSimulation) Disconnect(ctx context.Context, org, id string, version int64) error {
	return s.withStore(ctx, org, id, true, func(tx *gorm.DB, a StoreMemberAccess, store workbenchStoreRecord, row metaConnection) error {
		if store.Version != version {
			return ErrVersionConflict
		}
		if row.AttemptID == "" || row.Phase == "disconnected" {
			return ErrNotFound
		}
		row.Phase, row.KeyID, row.Ciphertext = "disconnected", "", ""
		if err := saveMetaConnection(tx, row); err != nil {
			return err
		}
		return updateConnectionRef(tx, store, "", a.ActorID, time.Now())
	})
}

func (s *MetaCatalogSimulation) publicationAccess(ctx context.Context, store workbenchStoreRecord, row metaConnection) (meta.Credentials, error) {
	now := time.Now()
	if !row.connected(store) || store.ServiceStatus == nil || *store.ServiceStatus != string(ServiceStatusActive) ||
		store.ServiceStartedAt == nil || store.ServiceExpiresAt == nil || now.Before(*store.ServiceStartedAt) || !now.Before(*store.ServiceExpiresAt) {
		return meta.Credentials{}, ErrOfficialConnectionUnavailable
	}
	cred, err := s.open(row)
	if err != nil {
		return cred, err
	}
	catalogs, _, err := s.provider.Choices(ctx, cred)
	if err != nil {
		return meta.Credentials{}, ErrOfficialConnectionUnavailable
	}
	binding := row.binding()
	for _, choice := range catalogs {
		if choice.ID == binding.CatalogID && choice.BusinessID == binding.BusinessID {
			return cred, nil
		}
	}
	return meta.Credentials{}, ErrOfficialConnectionUnavailable
}

func (s *MetaCatalogSimulation) PublicationBinding(ctx context.Context, org, id string) (MetaBinding, error) {
	var binding MetaBinding
	err := s.withStore(ctx, org, id, false, func(_ *gorm.DB, a StoreMemberAccess, store workbenchStoreRecord, row metaConnection) error {
		if !a.CanWrite && !a.Administrator {
			return ErrNotFound
		}
		if _, err := s.publicationAccess(ctx, store, row); err != nil {
			return err
		}
		binding = row.binding()
		return nil
	})
	return binding, err
}

// WithPublication holds the Store connection fence until the bounded send finishes.
func (s *MetaCatalogSimulation) WithPublication(ctx context.Context, org, id string, binding MetaBinding, product meta.Product,
	authorizeSend func(context.Context) (bool, error)) (remoteID string, err error) {
	err = s.withStore(ctx, org, id, false, func(_ *gorm.DB, a StoreMemberAccess, store workbenchStoreRecord, row metaConnection) error {
		if (!a.CanWrite && !a.Administrator) || row.binding() != binding {
			return ErrVersionConflict
		}
		cred, err := s.publicationAccess(ctx, store, row)
		if err != nil {
			return err
		}
		send, err := authorizeSend(ctx)
		if err != nil || !send {
			return err
		}
		remoteID, err = s.provider.Create(ctx, cred.Token, binding.CatalogID, product)
		return err
	})
	return remoteID, err
}

func (s *MetaCatalogSimulation) Observe(ctx context.Context, org, id string, expected MetaBinding, retailer, remoteID string) MetaObservation {
	out := MetaObservation{"unverified", "unverified", "unverified"}
	_ = s.withStore(ctx, org, id, false, func(_ *gorm.DB, a StoreMemberAccess, store workbenchStoreRecord, row metaConnection) error {
		binding := row.binding()
		if binding.CatalogID != expected.CatalogID || binding.BusinessID != expected.BusinessID {
			return ErrNotFound
		}
		cred, err := s.publicationAccess(ctx, store, row)
		if err != nil {
			return err
		}
		if found, err := s.provider.ReadProduct(ctx, cred.Token, expected.CatalogID, remoteID, retailer); err == nil && found {
			out.Catalog = "found"
		}
		if expected.WhatsAppID == "" {
			out.WhatsApp = "not_selected"
			return nil
		}
		if !cred.WhatsApp || binding.WhatsAppID != expected.WhatsAppID {
			return nil
		}
		_, accounts, err := s.provider.Choices(ctx, cred)
		if err != nil {
			return err
		}
		for _, account := range accounts {
			if account.ID != binding.WhatsAppID || account.BusinessID != binding.BusinessID {
				continue
			}
			linked, err := s.provider.WhatsAppLinked(ctx, cred.Token, account.ID, expected.CatalogID)
			if err == nil {
				out.WhatsApp = "different"
				if linked {
					out.WhatsApp = "matched"
				}
			}
			break
		}
		return nil
	})
	return out
}
