package pgschema

import (
	"encoding/json"
	"os"
	"testing"
)

func TestSchemaChecksFromLogicalRestore(t *testing.T) {
	// Catalog DDL only, without business rows or credentials. Captured from
	// PostgreSQL 17 before/after pg_dump + pg_restore across the trial owners.
	data, err := os.ReadFile("testdata/schema_restore_checks.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct{ Name, Original, Restored, OriginalPretty, RestoredPretty string }
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			for _, pair := range [][2]string{{c.Original, c.Restored}, {c.OriginalPretty, c.RestoredPretty}} {
				if !SchemaConstraintEqual(pair[0], pair[1]) {
					t.Fatalf("restore changed representation: %s\n%s", pair[0], pair[1])
				}
			}
			if SchemaConstraintEqual("CHECK (true)", c.Original) {
				t.Fatal("relaxed constraint admitted")
			}
		})
	}
}

func TestSchemaConstraintLogicalRestore(t *testing.T) {
	original := `CHECK (((state)::text = ANY ((ARRAY['prepared'::character varying, 'publishing'::character varying])::text[])))`
	restored := `CHECK (((state)::text = ANY (ARRAY[('prepared'::character varying)::text, ('publishing'::character varying)::text])))`
	if !SchemaConstraintEqual(restored, original) || !SchemaConstraintEqual(original, restored) {
		t.Fatal("logical restore must preserve the exact enum contract")
	}
	for _, different := range []string{
		`CHECK (true)`,
		`CHECK (((state)::text = ANY (ARRAY['prepared'::text, 'publishing'::text, 'failed'::text])))`,
		`CHECK (((state)::text <> ANY (ARRAY['prepared'::text, 'publishing'::text])))`,
		`CHECK (((other)::text = ANY (ARRAY['prepared'::text, 'publishing'::text])))`,
		`CHECK (((state)::text = ANY ((ARRAY['prepared'::varchar(1), 'publishing'::varchar(1)])::text[])))`,
		`CHECK (((state)::text = ANY ((ARRAY['prepared'::char, 'publishing'::char])::text[])))`,
		`CHECK (((state)::text = ANY (ARRAY['PREPARED'::text, 'publishing'::text])))`,
		`CHECK (((state)::text = ANY (ARRAY['prepared'::text, NULL::text])))`,
		`CHECK (((state)::text = ANY (ARRAY['prepared'::text, 'publishing'::text]))) NO INHERIT`,
		`CHECK (octet_length(command) <= 2097153)`,
		`CHECK (`,
		``,
	} {
		if SchemaConstraintEqual(different, original) {
			t.Errorf("different or incomplete contract accepted: %s", different)
		}
	}
}

func TestSchemaDefinitionPreservesLiteralsAndOtherCasts(t *testing.T) {
	for _, pair := range [][2]string{
		{`CHECK (amount::numeric(4,1) > 0)`, `CHECK (amount::numeric(5,1) > 0)`},
		{`CHECK (length(value) <= 64)`, `CHECK (length(value) <= 65)`},
		{`CHECK (value='a  b')`, `CHECK (value='a b')`},
		{`CHECK (value='(x)::text')`, `CHECK (value='x')`},
		{`FOREIGN KEY (id) REFERENCES x(id)`, `FOREIGN KEY (id) REFERENCES y(id)`},
		{`PRIMARY KEY (org,id)`, `PRIMARY KEY (id,org)`},
	} {
		if SchemaConstraintEqual(pair[0], pair[1]) {
			t.Errorf("contract changed: %v", pair)
		}
	}
}

func TestSchemaIndexLogicalRestore(t *testing.T) {
	original := `CREATE UNIQUE INDEX organization_invitation_active_contact ON public.organization_member_invitations USING btree (project_id, organization_id, contact) WHERE ((state)::text = ANY ((ARRAY['pending'::character varying, 'accepting'::character varying])::text[]))`
	restored := `CREATE UNIQUE INDEX organization_invitation_active_contact ON public.organization_member_invitations USING btree (project_id, organization_id, contact) WHERE ((state)::text = ANY (ARRAY[('pending'::character varying)::text, ('accepting'::character varying)::text]))`
	if !SchemaIndexEqual(original, restored) {
		t.Fatal("partial index changed only by logical restore")
	}
	for _, different := range []string{
		`CREATE INDEX organization_invitation_active_contact ON public.organization_member_invitations USING btree (project_id, organization_id, contact) WHERE ((state)::text = ANY (ARRAY['pending'::text,'accepting'::text]))`,
		`CREATE UNIQUE INDEX organization_invitation_active_contact ON public.organization_member_invitations USING btree (project_id, organization_id, contact) WHERE true`,
		`CREATE UNIQUE INDEX organization_invitation_active_contact ON public.organization_member_invitations USING btree (organization_id, project_id, contact) WHERE ((state)::text = ANY (ARRAY['pending'::text,'accepting'::text]))`,
		`CREATE UNIQUE INDEX organization_invitation_active_contact ON public.organization_member_invitations USING btree (project_id, organization_id, contact) WHERE ((state)::text = ANY (ARRAY['pending'::text,'accepted'::text]))`,
	} {
		if SchemaIndexEqual(original, different) {
			t.Errorf("index contract changed: %s", different)
		}
	}
}
