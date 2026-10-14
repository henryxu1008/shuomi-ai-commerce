package agentconfigpersistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math"
	"regexp"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"time"
)

func (s *Store) ReadSkillAdmission(ctx context.Context, scope agent.Scope) (agentconfig.SkillAdmission, error) {
	if !scopeOK(scope) {
		return agentconfig.SkillAdmission{}, agentconfig.ErrInvalid
	}
	if e := VerifySkillAdmissionSchema(ctx, s.db); e != nil {
		return agentconfig.SkillAdmission{}, e
	}
	var row struct {
		AllowPersonal bool
		Revision      int64
	}
	q := s.db.WithContext(ctx).Raw("SELECT allow_personal,revision FROM agent_configuration.skill_admissions WHERE organization_id=? AND agent_id=?", scope.OrganizationID, agentconfig.ContentAgentID).Scan(&row)
	if q.Error != nil {
		return agentconfig.SkillAdmission{}, q.Error
	}
	return agentconfig.SkillAdmission{AllowPersonal: row.AllowPersonal, Revision: decimal(row.Revision)}, nil
}
func (s *Store) SetSkillAdmission(ctx context.Context, c agentconfig.SkillAdmissionCommand, check func(context.Context) error) (agentconfig.SkillAdmission, error) {
	var result agentconfig.SkillAdmission
	if !scopeOK(c.Scope) || !agentconfig.UUID(c.Key) || c.Expected >= math.MaxInt64 || check == nil {
		return result, agentconfig.ErrInvalid
	}
	if e := VerifySkillAdmissionSchema(ctx, s.db); e != nil {
		return result, e
	}
	fp, e := hash(struct {
		Operation string
		Command   agentconfig.SkillAdmissionCommand
	}{"skill-admission", c})
	if e != nil {
		return result, e
	}
	e = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := check(ctx); e != nil {
			return e
		}
		now := time.Now().UTC()
		claim := commandRow{OrganizationID: c.Scope.OrganizationID, ActorID: c.Scope.ActorID, IdempotencyKey: c.Key, CommandID: uuid.NewString(), Operation: "skill-admission", AgentID: agentconfig.ContentAgentID, Fingerprint: fp, Receipt: []byte("{}"), CommittedAt: now}
		written := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&claim)
		if written.Error != nil {
			return written.Error
		}
		if written.RowsAffected == 0 {
			var old commandRow
			if e := tx.Where("organization_id=? AND actor_id=? AND idempotency_key=?", c.Scope.OrganizationID, c.Scope.ActorID, c.Key).Take(&old).Error; e != nil {
				return e
			}
			if old.Fingerprint != fp || old.Operation != "skill-admission" {
				return agentconfig.ErrConflict
			}
			if json.Unmarshal(old.Receipt, &result) != nil {
				return agentconfig.ErrUnavailable
			}
			return nil
		}
		if _, e := findAgent(tx, c.Scope.OrganizationID, agentconfig.ContentAgentID, true); e != nil {
			return e
		}
		var current struct {
			AllowPersonal bool
			Revision      int64
		}
		q := tx.Raw("SELECT allow_personal,revision FROM agent_configuration.skill_admissions WHERE organization_id=? AND agent_id=?", c.Scope.OrganizationID, agentconfig.ContentAgentID).Scan(&current)
		if q.Error != nil {
			return q.Error
		}
		if uint64(current.Revision) != c.Expected {
			return agentconfig.ErrRevision
		}
		after := current.Revision
		changed := current.AllowPersonal != c.AllowPersonal
		if changed {
			after++
			if q.RowsAffected == 0 {
				if e := tx.Exec("INSERT INTO agent_configuration.skill_admissions(organization_id,agent_id,allow_personal,revision,updated_by,updated_at) VALUES(?,?,?,?,?,?)", c.Scope.OrganizationID, agentconfig.ContentAgentID, c.AllowPersonal, after, c.Scope.ActorID, now).Error; e != nil {
					return e
				}
			} else if e := tx.Exec("UPDATE agent_configuration.skill_admissions SET allow_personal=?,revision=?,updated_by=?,updated_at=? WHERE organization_id=? AND agent_id=?", c.AllowPersonal, after, c.Scope.ActorID, now, c.Scope.OrganizationID, agentconfig.ContentAgentID).Error; e != nil {
				return e
			}
		}
		result = agentconfig.SkillAdmission{AllowPersonal: c.AllowPersonal, Revision: decimal(after)}
		raw, _ := json.Marshal(result)
		return tx.Model(&commandRow{}).Where("command_id=?", claim.CommandID).Updates(map[string]any{"receipt": raw, "before_revision": current.Revision, "after_revision": after}).Error
	})
	return result, e
}

const skillAdmissionDDL = `CREATE TABLE IF NOT EXISTS agent_configuration.skill_admissions(organization_id varchar(128) NOT NULL,agent_id varchar(128) NOT NULL CHECK(agent_id='product.content.agent'),allow_personal boolean NOT NULL,revision bigint NOT NULL CHECK(revision>0),updated_by varchar(128) NOT NULL,updated_at timestamptz NOT NULL,PRIMARY KEY(organization_id,agent_id),FOREIGN KEY(organization_id,agent_id) REFERENCES agent_configuration.organization_agents(organization_id,agent_id))`

// Initializer-only bounded policy grants; effective column/inherited authority
// is checked independently of Skill storage and original configuration grants.
func GrantSkillAdmissionRuntime(ctx context.Context, db *gorm.DB, role string) error {
	if !regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`).MatchString(role) {
		return agentconfig.ErrInvalid
	}
	if e := VerifySkillAdmissionSchema(ctx, db); e != nil {
		return e
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var safe bool
		if e := tx.Raw(`SELECT NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR pg_has_role(oid,'agent_configuration_owner','MEMBER')) FROM pg_roles WHERE rolname=?`, role).Scan(&safe).Error; e != nil || !safe {
			return agentconfig.ErrForbidden
		}
		quoted := `"` + role + `"`
		for _, query := range []string{"REVOKE ALL ON agent_configuration.skill_admissions FROM " + quoted, "GRANT SELECT,INSERT ON agent_configuration.skill_admissions TO " + quoted, "GRANT UPDATE(allow_personal,revision,updated_by,updated_at) ON agent_configuration.skill_admissions TO " + quoted} {
			if e := tx.Exec(query).Error; e != nil {
				return e
			}
		}
		var unsafe bool
		if e := tx.Raw(`SELECT has_schema_privilege(?,'agent_configuration','CREATE') OR has_table_privilege(?,'agent_configuration.skill_admissions','DELETE,TRUNCATE,UPDATE') OR EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='agent_configuration.skill_admissions'::regclass AND attnum>0 AND NOT attisdropped AND attname NOT IN ('allow_personal','revision','updated_by','updated_at') AND has_column_privilege(?,attrelid,attname,'UPDATE'))`, role, role, role).Scan(&unsafe).Error; e != nil || unsafe {
			return agentconfig.ErrForbidden
		}
		return nil
	})
}

func InstallSkillAdmissionSchema(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var owner string
		if e := tx.Raw("SELECT pg_get_userbyid(nspowner) FROM pg_namespace WHERE nspname='agent_configuration'").Scan(&owner).Error; e != nil || owner != "agent_configuration_owner" {
			return agentconfig.ErrUnavailable
		}
		for _, query := range []string{"SET LOCAL ROLE agent_configuration_owner", skillAdmissionDDL, "REVOKE ALL ON agent_configuration.skill_admissions FROM PUBLIC", "RESET ROLE"} {
			if e := tx.Exec(query).Error; e != nil {
				return e
			}
		}
		return VerifySkillAdmissionSchema(ctx, tx)
	})
}
func VerifySkillAdmissionSchema(ctx context.Context, db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return agentconfig.ErrUnavailable
	}
	var row struct{ Owner, Shape, Contract string }
	e := db.WithContext(ctx).Raw(`SELECT pg_get_userbyid(c.relowner) AS owner,(SELECT string_agg(a.attname||':'||format_type(a.atttypid,a.atttypmod)||':'||(NOT a.attnotnull)::text,'|' ORDER BY a.attname) FROM pg_attribute a WHERE a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped) AS shape,(SELECT string_agg(x.conname||':'||pg_get_constraintdef(x.oid)||':'||x.convalidated::text,'|' ORDER BY x.conname) FROM pg_constraint x WHERE x.conrelid=c.oid) AS contract FROM pg_class c WHERE c.oid=to_regclass('agent_configuration.skill_admissions') AND c.relkind='r'`).Scan(&row).Error
	if e != nil || row.Owner != "agent_configuration_owner" || row.Shape != "agent_id:character varying(128):false|allow_personal:boolean:false|organization_id:character varying(128):false|revision:bigint:false|updated_at:timestamp with time zone:false|updated_by:character varying(128):false" {
		return agentconfig.ErrUnavailable
	}
	digest := sha256.Sum256([]byte(row.Contract))
	if hex.EncodeToString(digest[:]) != "b207bf1e2bce9ebd9ae9cc868d5c0b4f4f60c8ef9cb78cf37e300c23e71611a4" {
		return agentconfig.ErrUnavailable
	}
	return nil
}
