ALTER TABLE ai_workbench.scheduled_plans ADD COLUMN IF NOT EXISTS report_spec jsonb;
ALTER TABLE ai_workbench.scheduled_plans ALTER COLUMN task_selection DROP NOT NULL;
ALTER TABLE ai_workbench.scheduled_plans DROP CONSTRAINT IF EXISTS scheduled_plans_kind_check;
ALTER TABLE ai_workbench.scheduled_plans DROP CONSTRAINT IF EXISTS scheduled_plans_budget_unit_check;
ALTER TABLE ai_workbench.scheduled_plans DROP CONSTRAINT IF EXISTS scheduled_plans_per_run_upper_bound_micros_check;
ALTER TABLE ai_workbench.scheduled_plans DROP CONSTRAINT IF EXISTS scheduled_plans_total_upper_bound_micros_check;
ALTER TABLE ai_workbench.scheduled_plans DROP CONSTRAINT IF EXISTS scheduled_report_capability;
ALTER TABLE ai_workbench.scheduled_plans ADD CONSTRAINT scheduled_report_capability CHECK (
 (kind='product.content.localize' AND task_selection IS NOT NULL AND report_spec IS NULL
  AND budget_unit='ai-micropoint-v1' AND per_run_upper_bound_micros>0 AND total_upper_bound_micros>0)
 OR (kind='store.operating-daily-report' AND task_selection IS NULL AND report_spec IS NOT NULL
  AND jsonb_typeof(report_spec)='object' AND report_spec - 'storeId' = '{"version":1,"periodPolicy":"previous_natural_day","businessZone":"Asia/Shanghai"}'::jsonb
  AND report_spec ? 'storeId' AND jsonb_typeof(report_spec->'storeId')='string' AND report_spec->>'storeId' ~ '^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$'
  AND report_spec->>'storeId'<>'00000000-0000-0000-0000-000000000000'
  AND execution_spec='{}'::jsonb AND budget_unit='none' AND per_run_upper_bound_micros=0 AND total_upper_bound_micros=0
  AND time_zone='Asia/Shanghai' AND (frequency='daily' OR (frequency='once' AND max_occurrences=1)))
);
CREATE TABLE IF NOT EXISTS ai_workbench.report_execution (
 occurrence_id uuid PRIMARY KEY, organization_id text NOT NULL, owner_user_id text NOT NULL,
 intent jsonb NOT NULL CHECK(jsonb_typeof(intent)='object' AND octet_length(intent::text) BETWEEN 2 AND 4096),
 fingerprint char(64) NOT NULL CHECK(fingerprint ~ '^[a-f0-9]{64}$'),
 capture_started boolean NOT NULL DEFAULT false, closed_no_write boolean NOT NULL DEFAULT false,
 report_id uuid, report_digest char(64), proof char(64),
 CHECK (NOT(capture_started AND closed_no_write)),
 CHECK ((report_id IS NULL AND report_digest IS NULL AND proof IS NULL) OR
   (capture_started AND report_id IS NOT NULL AND report_digest IS NOT NULL AND proof IS NOT NULL AND report_digest ~ '^[a-f0-9]{64}$' AND proof ~ '^[a-f0-9]{64}$')),
 FOREIGN KEY(occurrence_id,organization_id,owner_user_id) REFERENCES ai_workbench.scheduled_occurrences(id,organization_id,owner_user_id)
);
REVOKE ALL ON ai_workbench.report_execution FROM PUBLIC;
