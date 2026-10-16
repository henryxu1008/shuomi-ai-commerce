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

func TestSensitiveWordLegacyColumnMigrationsCoverLegacyStatusAndAuditColumns(t *testing.T) {
	t.Parallel()

	migrations := sensitiveWordLegacyColumnMigrations()

	status, ok := migrations["status"]
	if !ok {
		t.Fatal("expected status migration to exist")
	}
	if status.TargetType != "smallint" {
		t.Fatalf("status target type = %q, want smallint", status.TargetType)
	}
	if status.UsingExpression == "" {
		t.Fatal("expected status migration to include USING expression")
	}
	if !status.DropDefaultBeforeTypeChange {
		t.Fatal("expected status migration to drop legacy default before type change")
	}
	if status.DefaultExpression != "0" {
		t.Fatalf("status default expression = %q, want 0", status.DefaultExpression)
	}

	for _, column := range []string{"creator", "updater"} {
		migration, ok := migrations[column]
		if !ok {
			t.Fatalf("expected %s migration to exist", column)
		}
		if migration.TargetType != "varchar(128)" {
			t.Fatalf("%s target type = %q, want varchar(128)", column, migration.TargetType)
		}
		if migration.UsingExpression == "" {
			t.Fatalf("expected %s migration to include USING expression", column)
		}
	}
}

func TestPostgresColumnDefinitionNeedsTypeMigration(t *testing.T) {
	t.Parallel()

	if !postgresColumnDefinitionNeedsTypeMigration(
		postgresColumnDefinition{DataType: "character varying", CharacterMaximumLength: intPtr(20)},
		"smallint",
	) {
		t.Fatal("expected varchar(20) to require migration to smallint")
	}

	if !postgresColumnDefinitionNeedsTypeMigration(
		postgresColumnDefinition{DataType: "bigint"},
		"varchar(128)",
	) {
		t.Fatal("expected bigint to require migration to varchar(128)")
	}

	if postgresColumnDefinitionNeedsTypeMigration(
		postgresColumnDefinition{DataType: "character varying", CharacterMaximumLength: intPtr(128)},
		"varchar(128)",
	) {
		t.Fatal("expected varchar(128) to already match target type")
	}
}

func intPtr(value int) *int {
	return &value
}
