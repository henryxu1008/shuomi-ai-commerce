export function commercialResourcesFixture(organizationId = "org-B") {
  const updated = "2026-09-28T00:00:00Z";
  return {
    schema_version: "organization-resource-balances-v1" as const, organization_id: organizationId, observed_at: updated,
    resources: [
      { resource_type: "store_renewal_period" as const, unit: "period" as const, state: "recorded" as const, available: "1", allocated: "0", reserved: "0", consumed: "0", debt: "0", updated_at: updated },
      { unit_version:"ai-micropoint-v1" as const, resource_type: "ai_point" as const, unit: "point" as const, state: "recorded" as const, available: "9007199254.740993", allocated: "0", reserved: "0.000012", consumed: "0.000003", debt: "0", updated_at: updated },
      { resource_type: "data_row" as const, unit: "row" as const, state: "not_recorded" as const, available: null, allocated: null, reserved: null, consumed: null, debt: null, updated_at: null },
    ],
  };
}
