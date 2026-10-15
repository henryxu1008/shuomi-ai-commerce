package productagentruntime

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestPrepareRejectsNonConnectableRuntimeRolesWithoutRepair(t *testing.T) {
	for _, role := range []string{"product_agent_runtime", "title_credential_writer", "product_review_runtime"} {
		t.Run(role, func(t *testing.T) {
			target, prepareTarget := "run", PrepareRun
			if role == "product_review_runtime" {
				target, prepareTarget = "review", PrepareReview
			}
			f := newPrepareFixture(t, target)
			_, err := prepareTarget(context.Background(), f.db)
			require.NoError(t, err)
			var original int
			require.NoError(t, f.db.Raw("SELECT rolconnlimit FROM pg_catalog.pg_roles WHERE rolname=?", role).Scan(&original).Error)
			require.NoError(t, f.db.Exec("ALTER ROLE "+quote(role)+" CONNECTION LIMIT 0").Error)
			t.Cleanup(func() {
				require.NoError(t, f.db.Exec(fmt.Sprintf("ALTER ROLE %s CONNECTION LIMIT %d", quote(role), original)).Error)
			})
			connection, loginErr := pgx.Connect(context.Background(), f.dsn+" dbname="+f.name+" user="+role+" password=synthetic-native-role")
			if connection != nil {
				require.NoError(t, connection.Close(context.Background()))
			}
			require.ErrorContains(t, loginErr, "too many connections for role")
			before := retainedSnapshot(t, f.db, []string{"public.preserved_fact"})
			_, err = prepareTarget(context.Background(), f.db)
			require.Error(t, err, "SET LOCAL ROLE cannot prove that a runtime can actually connect")
			require.Equal(t, before, retainedSnapshot(t, f.db, []string{"public.preserved_fact"}))
			var retained int
			require.NoError(t, f.db.Raw("SELECT rolconnlimit FROM pg_catalog.pg_roles WHERE rolname=?", role).Scan(&retained).Error)
			require.Zero(t, retained)
		})
	}
}

func TestPrepareKeepsFiniteConnectableRuntimeRoleLimit(t *testing.T) {
	f := newPrepareFixture(t, "run")
	_, err := PrepareRun(context.Background(), f.db)
	require.NoError(t, err)
	require.NoError(t, f.db.Exec("ALTER ROLE product_agent_runtime CONNECTION LIMIT 1").Error)
	t.Cleanup(func() { require.NoError(t, f.db.Exec("ALTER ROLE product_agent_runtime CONNECTION LIMIT -1").Error) })
	connection, err := pgx.Connect(context.Background(), f.dsn+" dbname="+f.name+" user=product_agent_runtime password=synthetic-native-role")
	require.NoError(t, err)
	defer connection.Close(context.Background())
	_, err = PrepareRun(context.Background(), f.db)
	require.NoError(t, err, "positive limits are allowed; preparation does not govern current connection occupancy")
}
