ALTER TABLE report_center.saved_reports ADD COLUMN IF NOT EXISTS scheduled_origin jsonb;
ALTER TABLE report_center.saved_reports DROP CONSTRAINT IF EXISTS saved_reports_kind_check;
ALTER TABLE report_center.saved_reports DROP CONSTRAINT IF EXISTS saved_report_daily_origin;
ALTER TABLE report_center.saved_reports ADD CONSTRAINT saved_reports_kind_check CHECK(kind IN ('TITLE_REVIEW','SHEIN_RECORD','SHEIN_TARGET_RECORD','STORE_DAILY_REPORT'));
ALTER TABLE report_center.saved_reports ADD CONSTRAINT saved_report_daily_origin CHECK (
 (kind<>'STORE_DAILY_REPORT' AND scheduled_origin IS NULL AND product_key<>'') OR
 (kind='STORE_DAILY_REPORT' AND product_key='' AND source_version='1' AND scheduled_origin IS NOT NULL
  AND jsonb_typeof(scheduled_origin)='object'
  AND scheduled_origin ?& ARRAY['organizationId','actorId','memberId','planId','planRevision','occurrenceId','scheduledAt','storeId','day','rendererVersion']
  AND scheduled_origin->>'organizationId'=organization_id AND scheduled_origin->>'actorId'=actor_id
  AND scheduled_origin->>'occurrenceId'=source_id::text AND scheduled_origin->>'storeId'=store_id
  AND scheduled_origin->>'rendererVersion'='1' AND octet_length(scheduled_origin::text)<=4096)
);
