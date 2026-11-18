GRANT SELECT ON public.ai_search_credential_versions TO title_credential_writer;
GRANT INSERT (organization_id, revision) ON public.ai_search_credential_versions TO title_credential_writer;
GRANT USAGE ON SCHEMA public TO model_route_config_writer;
GRANT SELECT, INSERT, UPDATE ON public.ai_web_search_configuration, public.ai_web_search_requests TO model_route_config_writer;
GRANT SELECT, INSERT ON public.ai_web_search_config_receipts TO model_route_config_writer;
