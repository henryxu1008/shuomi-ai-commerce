package listingadmin

import (
	"strings"
	"testing"
)

func TestOwnerUserIDConstraintSQLRejectsBlankValuesAndIsIdempotent(t *testing.T) {
	name, statement := ownerUserIDConstraintSQL("listing_store")
	if name != "ck_listing_store_owner_user_id_nonblank" {
		t.Fatalf("constraint name = %q, want ck_listing_store_owner_user_id_nonblank", name)
	}
	for _, fragment := range []string{
		"pg_constraint",
		"JOIN pg_namespace AS schema_row ON schema_row.oid = table_row.relnamespace",
		"AND schema_row.nspname = current_schema()",
		"ADD CONSTRAINT \"ck_listing_store_owner_user_id_nonblank\"",
		"CHECK (NULLIF(regexp_replace(owner_user_id::text, '[[:space:]]', '', 'g'), '') IS NOT NULL) NOT VALID",
	} {
		if !strings.Contains(statement, fragment) {
			t.Fatalf("constraint SQL missing %q: %s", fragment, statement)
		}
	}
}
