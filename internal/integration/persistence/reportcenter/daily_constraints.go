package reportcenterpersistence

// Native PostgreSQL definitions of the explicitly installed daily-report contract.
var dailyReportConstraints = map[string]string{
	"saved_reports_content_check": `CHECK ((octet_length(content) <= 131072))`,
	"saved_reports_pkey":          `PRIMARY KEY (id)`,
	"saved_reports_organization_id_actor_id_kind_source_id_sourc_key": `UNIQUE (organization_id, actor_id, kind, source_id, source_version)`,
	"saved_reports_id_organization_id_actor_id_key":                   `UNIQUE (id, organization_id, actor_id)`,
	"saved_reports_kind_check":                                        `CHECK (((kind)::text = ANY ((ARRAY['TITLE_REVIEW'::character varying, 'SHEIN_RECORD'::character varying, 'SHEIN_TARGET_RECORD'::character varying, 'STORE_DAILY_REPORT'::character varying])::text[])))`,
	"saved_report_daily_origin":                                       `CHECK (((((kind)::text <> 'STORE_DAILY_REPORT'::text) AND (scheduled_origin IS NULL) AND ((product_key)::text <> ''::text)) OR (((kind)::text = 'STORE_DAILY_REPORT'::text) AND ((product_key)::text = ''::text) AND ((source_version)::text = '1'::text) AND (scheduled_origin IS NOT NULL) AND (jsonb_typeof(scheduled_origin) = 'object'::text) AND (scheduled_origin ?& ARRAY['organizationId'::text, 'actorId'::text, 'memberId'::text, 'planId'::text, 'planRevision'::text, 'occurrenceId'::text, 'scheduledAt'::text, 'storeId'::text, 'day'::text, 'rendererVersion'::text]) AND ((scheduled_origin ->> 'organizationId'::text) = (organization_id)::text) AND ((scheduled_origin ->> 'actorId'::text) = (actor_id)::text) AND ((scheduled_origin ->> 'occurrenceId'::text) = (source_id)::text) AND ((scheduled_origin ->> 'storeId'::text) = (store_id)::text) AND ((scheduled_origin ->> 'rendererVersion'::text) = '1'::text) AND (octet_length((scheduled_origin)::text) <= 4096))))`,
}
