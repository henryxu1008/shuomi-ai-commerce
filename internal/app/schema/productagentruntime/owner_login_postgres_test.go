package productagentruntime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreparePreservesOfficialLoginOwners(t *testing.T) {
	for _, target := range []string{"run", "review"} {
		t.Run(target, func(t *testing.T) {
			owner, prepareTarget := "product_agent_owner", PrepareRun
			if target == "review" {
				owner, prepareTarget = "acquisition_owner", PrepareReview
			}
			f := newPrepareFixture(t, target)
			// The official business-db bootstrap creates LOGIN schema owners.
			// Existing schema commands also authenticate as those same owners.
			require.NoError(t, f.db.Exec("ALTER ROLE "+quote(owner)+" LOGIN PASSWORD 'synthetic-native-role'").Error)
			t.Cleanup(func() {
				require.NoError(t, f.db.Exec("ALTER ROLE "+quote(owner)+" NOLOGIN PASSWORD NULL").Error)
			})
			connection := f.role(t, owner)
			var currentUser string
			require.NoError(t, connection.Raw("SELECT current_user").Scan(&currentUser).Error)
			require.Equal(t, owner, currentUser)
			before := retainedSnapshot(t, f.db, []string{"public.preserved_fact"})
			_, err := prepareTarget(context.Background(), connection)
			require.NoError(t, err, "native preparation must accept the current official owner login contract")
			require.Equal(t, before, retainedSnapshot(t, f.db, []string{"public.preserved_fact"}))
			var canLogin bool
			require.NoError(t, f.db.Raw("SELECT rolcanlogin FROM pg_catalog.pg_roles WHERE rolname=?", owner).Scan(&canLogin).Error)
			require.True(t, canLogin, "preparation must preserve the original owner's login capability")
			var userAfter string
			require.NoError(t, f.role(t, owner).Raw("SELECT current_user").Scan(&userAfter).Error)
			require.Equal(t, owner, userAfter, "the original password must still authenticate a fresh owner connection")
			ready, err := prepareTarget(context.Background(), connection)
			require.NoError(t, err)
			require.Equal(t, "ALREADY_READY", ready.Outcome)
		})
	}
}

func TestPrepareRejectsUnboundedLoginOwnersWithoutRepair(t *testing.T) {
	for _, target := range []string{"run", "review"} {
		for _, attribute := range []string{"SUPERUSER", "CREATEDB", "CREATEROLE", "REPLICATION", "BYPASSRLS"} {
			t.Run(target+"/"+attribute, func(t *testing.T) {
				owner, prepareTarget := "product_agent_owner", PrepareRun
				if target == "review" {
					owner, prepareTarget = "acquisition_owner", PrepareReview
				}
				f := newPrepareFixture(t, target)
				require.NoError(t, f.db.Exec("ALTER ROLE "+quote(owner)+" LOGIN "+attribute).Error)
				t.Cleanup(func() {
					require.NoError(t, f.db.Exec("ALTER ROLE "+quote(owner)+" NOLOGIN NO"+attribute).Error)
				})
				var attributesBefore string
				const attributesQuery = "SELECT json_build_array(rolcanlogin,rolsuper,rolcreatedb,rolcreaterole,rolreplication,rolbypassrls)::text FROM pg_catalog.pg_roles WHERE rolname=?"
				require.NoError(t, f.db.Raw(attributesQuery, owner).Scan(&attributesBefore).Error)
				before := retainedSnapshot(t, f.db, []string{"public.preserved_fact"})
				_, err := prepareTarget(context.Background(), f.db)
				require.Error(t, err, "LOGIN is valid but broad cluster authority remains rejected")
				require.Equal(t, before, retainedSnapshot(t, f.db, []string{"public.preserved_fact"}))
				var attributesAfter string
				require.NoError(t, f.db.Raw(attributesQuery, owner).Scan(&attributesAfter).Error)
				require.Equal(t, attributesBefore, attributesAfter, "rejection must not repair existing owner attributes")
			})
		}
	}
}
