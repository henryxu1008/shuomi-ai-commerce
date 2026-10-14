import { podRequest } from "./pod";
import { collectionID } from "../contracts/product-collection";
import {
  diagnosisInspectionSchema,
  diagnosisInputSchema,
  diagnosisIntentSchema,
  diagnosisPreparedSchema,
  diagnosisRunSchema,
  type DiagnosisIntent,
} from "../contracts/shop-diagnosis";
import type { ConfigurationScope } from "./agent-configuration";
export const inspectDiagnosis = (
  scope: ConfigurationScope,
  input: unknown,
  signal?: AbortSignal,
) =>
  podRequest(
    scope,
    "inspect",
    diagnosisInspectionSchema,
    signal,
    { key: crypto.randomUUID(), body: diagnosisInputSchema.parse(input) },
    "shop-diagnosis",
  );
export const readDiagnosisRun = (
  scope: ConfigurationScope,
  context: string,
  key: string,
  signal?: AbortSignal,
) =>
  podRequest(
    scope,
    "contexts/" +
      collectionID.parse(context) +
      "/runs/" +
      collectionID.parse(key),
    diagnosisRunSchema,
    signal,
    undefined,
    "shop-diagnosis",
  );
export const prepareDiagnosis = (
  scope: ConfigurationScope,
  intent: Extract<DiagnosisIntent, { kind: "prepare" }>,
  signal?: AbortSignal,
) =>
  podRequest(
    scope,
    "prepare",
    diagnosisPreparedSchema,
    signal,
    diagnosisIntentSchema.parse(intent),
    "shop-diagnosis",
  );
export function executeDiagnosis(
  scope: ConfigurationScope,
  intent: Exclude<DiagnosisIntent, { kind: "prepare" }>,
  signal?: AbortSignal,
) {
  const i = diagnosisIntentSchema.parse(intent);
  if (i.kind === "prepare") throw new Error("Invalid execution");
  return podRequest(
    scope,
    "contexts/" +
      i.contextId +
      "/runs" +
      (i.kind === "resume" ? "/" + i.key + "/resume" : ""),
    diagnosisRunSchema,
    signal,
    i,
    "shop-diagnosis",
  );
}
