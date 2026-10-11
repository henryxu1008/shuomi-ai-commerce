package storecenter

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const MaxMetadataVersion int64 = 9007199254740991

// Responsible is business ownership only. This port validates the canonical
// current membership; it has no grant-writing capability.
type ResponsibleMemberReader interface {
	ValidateResponsible(context.Context, string, string) error
}

type StoreProfile struct {
	OrganizationID      string    `json:"-" gorm:"column:organization_id;primaryKey;size:200;not null"`
	StoreID             string    `json:"storeId" gorm:"column:store_id;primaryKey;type:char(36);not null"`
	GroupID             *string   `json:"groupId" gorm:"column:group_id;type:char(36)"`
	ResponsibleMemberID *string   `json:"responsibleMemberId" gorm:"column:responsible_member_id;size:200"`
	LowStockThreshold   *int64    `json:"lowStockThreshold" gorm:"column:low_stock_threshold"`
	Version             int64     `json:"profileVersion" gorm:"column:version;not null"`
	UpdatedBy           string    `json:"-" gorm:"column:updated_by;size:200;not null"`
	UpdatedAt           time.Time `json:"updatedAt" gorm:"column:updated_at;not null;autoUpdateTime:false"`
}

func (StoreProfile) TableName() string { return "workbench_store_profiles" }

func (p StoreProfile) MarshalJSON() ([]byte, error) {
	type plain StoreProfile
	if p.Version > 0 {
		return json.Marshal(plain(p))
	}
	return json.Marshal(struct {
		plain
		UpdatedAt *time.Time `json:"updatedAt"`
	}{plain: plain(p)})
}

type StoreGroup struct {
	OrganizationID string     `json:"-" gorm:"column:organization_id;primaryKey;size:200;not null"`
	ID             string     `json:"id" gorm:"column:id;primaryKey;type:char(36);not null"`
	Name           string     `json:"name" gorm:"column:name;not null"`
	Version        int64      `json:"groupVersion" gorm:"column:version;not null"`
	UpdatedBy      string     `json:"-" gorm:"column:updated_by;size:200;not null"`
	UpdatedAt      time.Time  `json:"updatedAt" gorm:"column:updated_at;not null;autoUpdateTime:false"`
	DeletedAt      *time.Time `json:"-" gorm:"column:deleted_at"`
}

func (StoreGroup) TableName() string { return "workbench_store_groups" }

type ProfileTarget struct {
	StoreID string `json:"storeId"`
	Version int64  `json:"profileVersion"`
}
type MetadataCommand struct {
	OperationID         string          `json:"-"`
	Action              string          `json:"action"`
	Name                string          `json:"name"`
	GroupID             *string         `json:"groupId"`
	GroupVersion        int64           `json:"groupVersion"`
	Stores              []ProfileTarget `json:"stores"`
	ResponsibleMemberID *string         `json:"responsibleMemberId"`
	LowStockThreshold   *int64          `json:"lowStockThreshold"`
}
type MetadataResult struct {
	OperationID string         `json:"operationId"`
	Profiles    []StoreProfile `json:"profiles"`
	Group       *StoreGroup    `json:"group"`
	Replayed    bool           `json:"replayed"`
}

// This append-only command journal is also the metadata audit trail. The
// canonical safe input and frozen result commit with the mutation.
type metadataOperation struct {
	OrganizationID string    `gorm:"column:organization_id;primaryKey;size:200;not null"`
	OperationID    string    `gorm:"column:operation_id;primaryKey;type:char(36);not null"`
	ActorID        string    `gorm:"column:actor_id;size:200;not null"`
	MemberID       string    `gorm:"column:member_id;size:200;not null"`
	Action         string    `gorm:"column:action;size:64;not null"`
	Fingerprint    string    `gorm:"column:fingerprint;size:64;not null"`
	CommandJSON    string    `gorm:"column:command_json;not null"`
	ResultJSON     string    `gorm:"column:result_json;not null"`
	CreatedAt      time.Time `gorm:"column:created_at;not null"`
}

func (metadataOperation) TableName() string { return "workbench_store_metadata_operations" }

type MetadataRepository struct {
	scoped  *MemberScopedStoreRepository
	members ResponsibleMemberReader
}

func NewMetadataRepository(db *gorm.DB, access StoreMemberAuthorizer, members ResponsibleMemberReader) (*MetadataRepository, error) {
	scoped, err := NewMemberScopedStoreRepository(db, access)
	if err != nil {
		return nil, err
	}
	return &MetadataRepository{scoped: scoped, members: members}, nil
}
func (r *MetadataRepository) Profile(ctx context.Context, org, id string) (StoreProfile, error) {
	if _, err := r.scoped.Get(ctx, org, id); err != nil {
		return StoreProfile{}, err
	}
	var profile StoreProfile
	err := r.scoped.db.WithContext(ctx).Where("organization_id=? AND store_id=?", org, id).Take(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		profile = StoreProfile{OrganizationID: org, StoreID: id}
		err = nil
	}
	if err == nil {
		_, err = r.scoped.Get(ctx, org, id)
	}
	return profile, err
}
func (r *MetadataRepository) Groups(ctx context.Context, org string) ([]StoreGroup, error) {
	if _, err := r.scoped.authorize(ctx, org, false); err != nil {
		return nil, err
	}
	groups := []StoreGroup{}
	err := r.scoped.db.WithContext(ctx).Where("organization_id=? AND deleted_at IS NULL", org).Order("name ASC,id ASC").Find(&groups).Error
	if err == nil {
		_, err = r.scoped.authorize(ctx, org, false)
	}
	return groups, err
}

var ErrMetadataInvalid = errors.New("invalid store metadata command")
var ErrMetadataKeyConflict = errors.New("store metadata operation key conflict")

func normalizeMetadata(command MetadataCommand) (MetadataCommand, error) {
	if _, err := canonicalUUID(command.OperationID); err != nil {
		return command, ErrMetadataInvalid
	}
	if command.GroupID != nil {
		if _, err := canonicalUUID(*command.GroupID); err != nil {
			return command, ErrMetadataInvalid
		}
	}
	if command.GroupVersion < 0 || command.GroupVersion >= MaxMetadataVersion || len(command.Stores) > 100 {
		return command, ErrMetadataInvalid
	}
	command.Stores = append([]ProfileTarget{}, command.Stores...)
	sort.Slice(command.Stores, func(i, j int) bool { return command.Stores[i].StoreID < command.Stores[j].StoreID })
	for i, target := range command.Stores {
		if _, err := canonicalUUID(target.StoreID); err != nil || target.Version < 0 || target.Version >= MaxMetadataVersion || i > 0 && command.Stores[i-1].StoreID == target.StoreID {
			return command, ErrMetadataInvalid
		}
	}
	switch command.Action {
	case "create_group", "rename_group", "delete_group":
		if len(command.Stores) > 0 || command.ResponsibleMemberID != nil || command.LowStockThreshold != nil || command.Action == "create_group" && (command.GroupID != nil || command.GroupVersion != 0) || command.Action != "create_group" && command.GroupID == nil {
			return command, ErrMetadataInvalid
		}
		if command.Action == "delete_group" {
			if command.Name != "" {
				return command, ErrMetadataInvalid
			}
			break
		}
		command.Name = strings.TrimSpace(command.Name)
		if !utf8.ValidString(command.Name) || utf8.RuneCountInString(command.Name) < 1 || utf8.RuneCountInString(command.Name) > 60 {
			return command, ErrMetadataInvalid
		}
		for _, char := range command.Name {
			if unicode.IsControl(char) {
				return command, ErrMetadataInvalid
			}
		}
	case "set_profile", "assign_group":
		if len(command.Stores) == 0 || command.Name != "" || command.GroupVersion != 0 || command.Action == "set_profile" && len(command.Stores) != 1 || command.Action == "assign_group" && (command.ResponsibleMemberID != nil || command.LowStockThreshold != nil) {
			return command, ErrMetadataInvalid
		}
		if command.ResponsibleMemberID != nil {
			if _, err := validateOpaqueIdentity("member", *command.ResponsibleMemberID, MaxSubjectBytes); err != nil {
				return command, ErrMetadataInvalid
			}
		}
		if command.LowStockThreshold != nil && (*command.LowStockThreshold < 0 || *command.LowStockThreshold > MaxMetadataVersion) {
			return command, ErrMetadataInvalid
		}
	default:
		return command, ErrMetadataInvalid
	}
	return command, nil
}
func metadataGroupAction(action string) bool {
	return action == "create_group" || action == "rename_group" || action == "delete_group"
}
func metadataAuthorize(tx *gorm.DB, access StoreMemberAccess, command MetadataCommand) error {
	if metadataGroupAction(command.Action) && !access.Administrator {
		return ErrNotFound
	}
	for _, target := range command.Stores {
		if err := lockMemberStore(tx, access.OrganizationID, target.StoreID); err != nil {
			return err
		}
		if err := requireMemberGrant(tx, access, target.StoreID); err != nil {
			return err
		}
	}
	return nil
}
func (r *MetadataRepository) Execute(ctx context.Context, org string, command MetadataCommand) (MetadataResult, error) {
	command, err := normalizeMetadata(command)
	if err != nil {
		return MetadataResult{}, err
	}
	access, err := r.scoped.authorize(ctx, org, true)
	if err != nil {
		return MetadataResult{}, err
	}
	raw, _ := json.Marshal(command)
	fingerprint := hashTuple("store-metadata-v1", org, string(raw))
	result := MetadataResult{OperationID: command.OperationID, Profiles: []StoreProfile{}}
	err = r.scoped.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize an operation before group/Store locks, including group creation.
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", org+":"+command.OperationID).Error; err != nil {
				return err
			}
		}
		var group StoreGroup
		groupErr := gorm.ErrRecordNotFound
		if command.GroupID != nil {
			groupErr = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND id=?", org, *command.GroupID).Take(&group).Error
			if groupErr != nil && !errors.Is(groupErr, gorm.ErrRecordNotFound) {
				return groupErr
			}
		}
		if err := metadataAuthorize(tx, access, command); err != nil {
			return err
		}
		var receipt metadataOperation
		err := tx.Where("organization_id=? AND operation_id=?", org, command.OperationID).Take(&receipt).Error
		if err == nil {
			if receipt.ActorID != access.ActorID || receipt.MemberID != access.MemberID || receipt.Action != command.Action || receipt.Fingerprint != fingerprint {
				return ErrMetadataKeyConflict
			}
			if json.Unmarshal([]byte(receipt.ResultJSON), &result) != nil {
				return ErrDependencyUnavailable
			}
			result.Replayed = true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if command.GroupID != nil && (groupErr != nil || group.DeletedAt != nil) {
			return ErrNotFound
		}
		at := time.Now().UTC()
		if metadataGroupAction(command.Action) {
			if command.Action == "create_group" {
				group = StoreGroup{OrganizationID: org, ID: uuid.NewString(), Name: command.Name, Version: 1, UpdatedBy: access.ActorID, UpdatedAt: at}
				if err = tx.Create(&group).Error; err != nil {
					return err
				}
			} else {
				if group.Version != command.GroupVersion || group.Version >= MaxMetadataVersion {
					return ErrVersionConflict
				}
				if command.Action == "delete_group" {
					profiles := []StoreProfile{}
					if err = tx.Where("organization_id=? AND group_id=?", org, group.ID).Order("store_id ASC").Find(&profiles).Error; err != nil {
						return err
					}
					// The group lock prevents new assignments. Lock active and deleted Store
					// rows in the same global order, then profile rows; no Store is removed.
					for _, profile := range profiles {
						var store workbenchStoreRecord
						if err = tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND id=?", org, profile.StoreID).Take(&store).Error; err != nil {
							return err
						}
						var locked StoreProfile
						if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=?", org, profile.StoreID).Take(&locked).Error; err != nil {
							return err
						}
						if locked.GroupID == nil || *locked.GroupID != group.ID {
							continue
						}
						if locked.Version >= MaxMetadataVersion {
							return ErrVersionConflict
						}
						if err = tx.Model(&locked).Updates(map[string]any{"group_id": nil, "version": locked.Version + 1, "updated_by": access.ActorID, "updated_at": at}).Error; err != nil {
							return err
						}
					}
					group.DeletedAt = &at
				} else {
					group.Name = command.Name
				}
				group.Version++
				group.UpdatedBy = access.ActorID
				group.UpdatedAt = at
				if err = tx.Model(&StoreGroup{}).Where("organization_id=? AND id=?", org, group.ID).Select("name", "version", "updated_by", "updated_at", "deleted_at").Updates(&group).Error; err != nil {
					return err
				}
			}
			result.Group = &group
		} else {

			for _, target := range command.Stores {
				var profile StoreProfile
				err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=?", org, target.StoreID).Take(&profile).Error
				fresh := errors.Is(err, gorm.ErrRecordNotFound)
				if err != nil && !fresh {
					return err
				}
				if profile.Version != target.Version {
					return ErrVersionConflict
				}

				if command.ResponsibleMemberID != nil && (profile.ResponsibleMemberID == nil || *profile.ResponsibleMemberID != *command.ResponsibleMemberID) {
					if isNilDependency(r.members) {
						return ErrDependencyUnavailable
					}
					if err = r.members.ValidateResponsible(ctx, org, *command.ResponsibleMemberID); err != nil {
						return err
					}
				}
				profile.OrganizationID = org
				profile.StoreID = target.StoreID
				profile.GroupID = command.GroupID
				if command.Action == "set_profile" {
					profile.ResponsibleMemberID = command.ResponsibleMemberID
					profile.LowStockThreshold = command.LowStockThreshold
				}
				profile.Version++
				profile.UpdatedBy = access.ActorID
				profile.UpdatedAt = at
				if fresh {
					err = tx.Create(&profile).Error
				} else {
					err = tx.Model(&StoreProfile{}).Where("organization_id=? AND store_id=?", org, target.StoreID).Select("group_id", "responsible_member_id", "low_stock_threshold", "version", "updated_by", "updated_at").Updates(&profile).Error
				}
				if err != nil {
					return err
				}
				result.Profiles = append(result.Profiles, profile)
			}
		}
		frozen, _ := json.Marshal(result)
		return tx.Create(&metadataOperation{OrganizationID: org, OperationID: command.OperationID, ActorID: access.ActorID, MemberID: access.MemberID, Action: command.Action, Fingerprint: fingerprint, CommandJSON: string(raw), ResultJSON: string(frozen), CreatedAt: at}).Error
	})
	return result, err
}
func (r *MetadataRepository) Receipt(ctx context.Context, org, key string) (MetadataResult, error) {
	if _, err := canonicalUUID(key); err != nil {
		return MetadataResult{}, ErrNotFound
	}
	access, err := r.scoped.authorize(ctx, org, true)
	if err != nil {
		return MetadataResult{}, err
	}
	var receipt metadataOperation
	if err = r.scoped.db.WithContext(ctx).Where("organization_id=? AND operation_id=?", org, key).Take(&receipt).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return MetadataResult{}, ErrNotFound
	} else if err != nil {
		return MetadataResult{}, err
	}
	if receipt.ActorID != access.ActorID || receipt.MemberID != access.MemberID {
		return MetadataResult{}, ErrNotFound
	}
	var command MetadataCommand
	if json.Unmarshal([]byte(receipt.CommandJSON), &command) != nil {
		return MetadataResult{}, ErrDependencyUnavailable
	}
	// Execute reauthorizes and resolves the original immutable receipt.
	command.OperationID = key
	return r.Execute(ctx, org, command)
}
