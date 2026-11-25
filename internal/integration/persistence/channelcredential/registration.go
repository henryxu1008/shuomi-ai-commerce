package channelcredentialstore

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	credential "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/channelcredential"
	"gorm.io/gorm"
)

const createRegistrationVersion = "create-channel-v1"
const renameRegistrationVersion = "rename-channel-v1"

func (s *Store) Registration(ctx context.Context, org, id string) (credential.Registration, error) {
	var result credential.Registration
	if s == nil || ctx == nil || ctx.Err() != nil || !credential.ValidID(org) || !credential.IsRegisteredConnection(id) {
		return result, credential.ErrInvalid
	}
	err := s.db.WithContext(ctx).Table("public.ai_channel_registrations").
		Where("organization_id=? AND connection_id=?", org, id).Take(&result).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, credential.ErrNotFound
	}
	if err != nil || result.OrganizationID != org || result.ConnectionID != id {
		return credential.Registration{}, credential.ErrUnavailable
	}
	return result, nil
}

func (s *Store) Registrations(ctx context.Context, org string) ([]credential.Registration, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || !credential.ValidID(org) {
		return nil, credential.ErrInvalid
	}
	result := []credential.Registration{}
	err := s.db.WithContext(ctx).Table("public.ai_channel_registrations").
		Where("organization_id=?", org).Order("connection_id").Limit(129).Find(&result).Error
	if err != nil || len(result) > 128 {
		return nil, credential.ErrUnavailable
	}
	return result, nil
}

// InstallRegistrations is an explicit additive Run DB owner operation.
func InstallRegistrations(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || ctx.Err() != nil || db == nil || db.Dialector.Name() != "postgres" {
		return credential.ErrUnavailable
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var owner bool
		err := tx.Raw(`SELECT current_user='product_agent_owner' AND current_schema()='public'
 AND EXISTS(SELECT 1 FROM pg_class WHERE oid=to_regclass('public.ai_channel_connections')
 AND relkind='r' AND relowner=(SELECT oid FROM pg_roles WHERE rolname=current_user))`).Scan(&owner).Error
		if err != nil || !owner {
			return credential.ErrUnavailable
		}
		return tx.Exec(`CREATE TABLE public.ai_channel_registrations (
 organization_id varchar(128) NOT NULL,
 connection_id varchar(128) NOT NULL,
 template varchar(128) NOT NULL CHECK (template IN ('openai-mini-v1','openai-compatible-mini-v1','grsai-product-image-v1')),
 name varchar(128) NOT NULL CHECK (length(name)>0),
 revision bigint NOT NULL CHECK (revision>0),
 PRIMARY KEY (organization_id, connection_id),
 FOREIGN KEY (organization_id, connection_id) REFERENCES public.ai_channel_connections(organization_id, connection_id)
)`).Error
	})
}

// EnableGRSaiImageRegistration widens only the known template constraint on
// existing managed registrations. It is called by the explicit Run DB owner.
func EnableGRSaiImageRegistration(ctx context.Context, db *gorm.DB) (bool, error) {
	return enableOwnedImageRegistration(ctx, db, false)
}

func EnableToAPIsImageRegistration(ctx context.Context, db *gorm.DB) (bool, error) {
	return enableOwnedImageRegistration(ctx, db, true)
}

func enableOwnedImageRegistration(ctx context.Context, db *gorm.DB, toapis bool) (bool, error) {
	if ctx == nil || ctx.Err() != nil || db == nil || db.Dialector.Name() != "postgres" {
		return false, credential.ErrUnavailable
	}
	var owner bool
	if err := db.WithContext(ctx).Raw(`SELECT current_user='product_agent_owner' AND current_schema()='public'`).Scan(&owner).Error; err != nil || !owner {
		return false, credential.ErrUnavailable
	}
	definition, err := registrationTemplateConstraint(ctx, db)
	if err != nil {
		return false, err
	}
	if definition == toapisImageRegistrationTemplates || definition == ownedImageRegistrationTemplates && !toapis {
		return false, nil
	}
	if definition != ownedImageRegistrationTemplates && definition != "CHECKtemplate=ANYARRAY['openai-mini-v1','openai-compatible-mini-v1']" {
		return false, credential.ErrUnavailable
	}
	templates := "'openai-mini-v1','openai-compatible-mini-v1','grsai-product-image-v1'"
	if toapis {
		templates += ",'toapis-product-image-v1'"
	}
	err = db.WithContext(ctx).Exec(`ALTER TABLE public.ai_channel_registrations
DROP CONSTRAINT ai_channel_registrations_template_check,
ADD CONSTRAINT ai_channel_registrations_template_check CHECK (template IN (` + templates + `))`).Error
	return err == nil, err
}

const ownedImageRegistrationTemplates = "CHECKtemplate=ANYARRAY['openai-mini-v1','openai-compatible-mini-v1','grsai-product-image-v1']"
const toapisImageRegistrationTemplates = "CHECKtemplate=ANYARRAY['openai-mini-v1','openai-compatible-mini-v1','grsai-product-image-v1','toapis-product-image-v1']"

// GRSaiImageRegistrationEnabled is a read-only preflight for the bounded
// application role. Schema edits remain with the explicit Run DB owner.
func GRSaiImageRegistrationEnabled(ctx context.Context, db *gorm.DB) bool {
	definition, err := registrationTemplateConstraint(ctx, db)
	return err == nil && (definition == ownedImageRegistrationTemplates || definition == toapisImageRegistrationTemplates)
}

func ToAPIsImageRegistrationEnabled(ctx context.Context, db *gorm.DB) bool {
	definition, err := registrationTemplateConstraint(ctx, db)
	return err == nil && definition == toapisImageRegistrationTemplates
}

func registrationTemplateConstraint(ctx context.Context, db *gorm.DB) (string, error) {
	if ctx == nil || ctx.Err() != nil || db == nil || db.Dialector.Name() != "postgres" {
		return "", credential.ErrUnavailable
	}
	var definition string
	q := db.WithContext(ctx).Raw(`SELECT pg_get_constraintdef(c.oid) FROM pg_constraint c
JOIN pg_class t ON t.oid=c.conrelid
WHERE t.oid=to_regclass('public.ai_channel_registrations') AND t.relkind='r' AND t.relpersistence='p'
AND pg_get_userbyid(t.relowner)='product_agent_owner'
AND c.conname='ai_channel_registrations_template_check' AND c.contype='c' AND c.convalidated AND NOT c.connoinherit
AND c.conkey=ARRAY[(SELECT attnum FROM pg_attribute WHERE attrelid=t.oid AND attname='template' AND NOT attisdropped)]`).Scan(&definition)
	if q.Error != nil || q.RowsAffected != 1 {
		return "", credential.ErrUnavailable
	}
	return strings.NewReplacer("::character varying", "", "::text[]", "", "::text", "", "(", "", ")", "", " ", "").Replace(definition), nil
}

func (s *Store) CreateRegistration(ctx context.Context, command credential.RegistrationCommand) (credential.RegistrationResult, error) {
	var result credential.RegistrationResult
	if command.Validate() != nil {
		return result, credential.ErrInvalid
	}
	if s == nil || s.protection == nil || ctx == nil || ctx.Err() != nil {
		return result, credential.ErrUnavailable
	}
	// Keep the representation explicit. Embedded identity fields must not
	// shadow each other in the serialized HMAC envelope.
	c := command.Command
	fingerprint, err := s.protection.fingerprint(createRegistrationVersion, struct {
		Connection   credential.Command
		Registration credential.Registration
	}{c, command.Registration}, c.Secret)
	if err != nil {
		return result, err
	}
	err = s.commandTransaction(ctx, c, fingerprint, createRegistrationVersion+":", func(tx *gorm.DB) (any, error) {
		if tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", "channel-registration-org:"+c.OrganizationID).Error != nil {
			return nil, credential.ErrUnavailable
		}
		var count int64
		if tx.Table("public.ai_channel_registrations").Where("organization_id=?", c.OrganizationID).Count(&count).Error != nil {
			return nil, credential.ErrUnavailable
		}
		if count >= 128 {
			return nil, credential.ErrRegistrationLimit
		}
		m, err := s.writeConnection(tx, c)
		if err != nil {
			return nil, err
		}
		if err := tx.Table("public.ai_channel_registrations").Create(&command.Registration).Error; err != nil {
			return nil, credential.ErrUnavailable
		}
		result = credential.NewRegistrationResult(command.Registration, m)
		return result, nil
	}, func(row ReceiptRow) error {
		var err error
		result, err = decodeRegistrationReceipt(row, c.OrganizationID)
		return err
	})
	if err != nil {
		return credential.RegistrationResult{}, err
	}
	return result, nil
}

func (s *Store) RegistrationReceipt(ctx context.Context, org, actor, key string) (credential.RegistrationResult, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || !credential.ValidID(org) || !credential.ValidID(actor) || !credential.ValidID(key) {
		return credential.RegistrationResult{}, credential.ErrUnavailable
	}
	row, err := commandReceipt(s.db.WithContext(ctx), org, actor, key, "", createRegistrationVersion+":")
	if err != nil {
		return credential.RegistrationResult{}, err
	}
	return decodeRegistrationReceipt(row, org)
}

func decodeRegistrationReceipt(row ReceiptRow, org string) (credential.RegistrationResult, error) {
	var result credential.RegistrationResult
	if len(row.Payload) > 16384 || json.Unmarshal(row.Payload, &result) != nil ||
		result.Registration.OrganizationID != org || result.Connection.OrganizationID != org ||
		!credential.IsRegisteredConnection(result.Registration.ConnectionID) ||
		result.Registration.ConnectionID != result.Connection.ConnectionID ||
		result.Registration.Revision != 1 || result.Connection.Revision != 1 ||
		!result.Connection.KeyPresent || !result.Connection.Enabled {
		return credential.RegistrationResult{}, credential.ErrUnavailable
	}
	return result, nil
}

func (s *Store) RenameRegistration(ctx context.Context, cmd credential.RenameRegistrationCommand) (credential.Registration, error) {
	var result credential.Registration
	if !credential.ValidID(cmd.OrganizationID) || !credential.ValidID(cmd.ActorID) || !credential.ValidID(cmd.RequestKey) ||
		!credential.IsRegisteredConnection(cmd.ConnectionID) || !credential.ValidID(cmd.Input.Name) ||
		cmd.Input.ExpectedRevision < 1 || cmd.Input.ExpectedRevision >= 1<<62 {
		return result, credential.ErrInvalid
	}
	if s == nil || s.protection == nil || ctx == nil || ctx.Err() != nil {
		return result, credential.ErrUnavailable
	}
	fingerprint, err := s.protection.fingerprint(renameRegistrationVersion, cmd, "")
	if err != nil {
		return result, err
	}
	key := credential.Command{OrganizationID: cmd.OrganizationID, ActorID: cmd.ActorID, Key: cmd.RequestKey, ConnectionID: cmd.ConnectionID}
	err = s.commandTransaction(ctx, key, fingerprint, renameRegistrationVersion+":", func(tx *gorm.DB) (any, error) {
		local := &Store{db: tx}
		var err error
		result, err = local.Registration(ctx, cmd.OrganizationID, cmd.ConnectionID)
		if err != nil {
			return nil, err
		}
		if result.Revision != cmd.Input.ExpectedRevision {
			return nil, credential.ErrConflict
		}
		result.Name, result.Revision = cmd.Input.Name, result.Revision+1
		write := tx.Table("public.ai_channel_registrations").Where("organization_id=? AND connection_id=? AND revision=?",
			cmd.OrganizationID, cmd.ConnectionID, cmd.Input.ExpectedRevision).Updates(map[string]any{"name": result.Name, "revision": result.Revision})
		if write.Error != nil || write.RowsAffected != 1 {
			return nil, credential.ErrUnavailable
		}
		return result, nil
	}, func(row ReceiptRow) error {
		var err error
		result, err = decodeRenameReceipt(row, cmd.OrganizationID)
		return err
	})
	if err != nil {
		return credential.Registration{}, err
	}
	return result, nil
}

func (s *Store) RenameReceipt(ctx context.Context, org, actor, key string) (credential.Registration, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || !credential.ValidID(org) || !credential.ValidID(actor) || !credential.ValidID(key) {
		return credential.Registration{}, credential.ErrUnavailable
	}
	row, err := commandReceipt(s.db.WithContext(ctx), org, actor, key, "", renameRegistrationVersion+":")
	if err != nil {
		return credential.Registration{}, err
	}
	return decodeRenameReceipt(row, org)
}

func decodeRenameReceipt(row ReceiptRow, org string) (credential.Registration, error) {
	var result credential.Registration
	if len(row.Payload) > 16384 || json.Unmarshal(row.Payload, &result) != nil ||
		result.OrganizationID != org || !credential.IsRegisteredConnection(result.ConnectionID) ||
		!credential.ValidID(result.Name) || result.Revision < 2 {
		return credential.Registration{}, credential.ErrUnavailable
	}
	return result, nil
}
