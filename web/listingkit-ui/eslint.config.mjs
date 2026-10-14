import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  {
    // Retired Task fields cannot enter current UI code. Historical test fixtures
    // remain exempt; the retired production helper/type exemptions are gone.
    files: ["src/**/*.{ts,tsx}"],
    ignores: [
      "src/**/*.test.ts",
      "src/**/*.test.tsx",
    ],
    rules: {
      "no-restricted-syntax": [
        "error",
        {
          selector:
            "MemberExpression[property.type='Identifier'][property.name='request_draft']",
          message:
            "Use draft_payload instead of request_draft in new UI code.",
        },
        {
          selector:
            "MemberExpression[computed=true][property.type='Literal'][property.value='request_draft']",
          message:
            "Use draft_payload instead of request_draft in new UI code.",
        },
        {
          selector:
            "Property[key.type='Identifier'][key.name='request_draft']",
          message:
            "Use draft_payload instead of request_draft in new UI code.",
        },
        {
          selector:
            "Property[key.type='Literal'][key.value='request_draft']",
          message:
            "Use draft_payload instead of request_draft in new UI code.",
        },
        {
          selector:
            "MemberExpression[property.type='Identifier'][property.name='preview_product']",
          message:
            "Use preview_payload instead of preview_product in new UI code.",
        },
        {
          selector:
            "MemberExpression[computed=true][property.type='Literal'][property.value='preview_product']",
          message:
            "Use preview_payload instead of preview_product in new UI code.",
        },
        {
          selector:
            "Property[key.type='Identifier'][key.name='preview_product']",
          message:
            "Use preview_payload instead of preview_product in new UI code.",
        },
        {
          selector:
            "Property[key.type='Literal'][key.value='preview_product']",
          message:
            "Use preview_payload instead of preview_product in new UI code.",
        },
        {
          selector:
            "MemberExpression[property.type='Identifier'][property.name='final_draft']",
          message:
            "Use final_submission_draft instead of final_draft in new UI code.",
        },
        {
          selector:
            "MemberExpression[computed=true][property.type='Literal'][property.value='final_draft']",
          message:
            "Use final_submission_draft instead of final_draft in new UI code.",
        },
        {
          selector:
            "Property[key.type='Identifier'][key.name='final_draft']",
          message:
            "Use final_submission_draft instead of final_draft in new UI code.",
        },
        {
          selector:
            "Property[key.type='Literal'][key.value='final_draft']",
          message:
            "Use final_submission_draft instead of final_draft in new UI code.",
        },
        {
          selector:
            "MemberExpression[property.type='Identifier'][property.name='sds_sync']",
          message:
            "Use sds_design_result instead of sds_sync in new UI code.",
        },
        {
          selector:
            "MemberExpression[computed=true][property.type='Literal'][property.value='sds_sync']",
          message:
            "Use sds_design_result instead of sds_sync in new UI code.",
        },
        {
          selector:
            "Property[key.type='Identifier'][key.name='sds_sync']",
          message:
            "Use sds_design_result instead of sds_sync in new UI code.",
        },
        {
          selector:
            "Property[key.type='Literal'][key.value='sds_sync']",
          message:
            "Use sds_design_result instead of sds_sync in new UI code.",
        },
      ],
    },
  },
  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
  ]),
]);

export default eslintConfig;
