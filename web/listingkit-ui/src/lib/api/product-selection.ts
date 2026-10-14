import { podRequest } from "./pod";
import { collectionID } from "../contracts/product-collection";
import {
  selectionInspectionSchema,
  selectionInputSchema,
  selectionIntentSchema,
  selectionPreparedSchema,
  selectionRunSchema,
  type SelectionIntent,
} from "../contracts/product-selection";
import type { ConfigurationScope } from "./agent-configuration";
export const inspectSelection = (
  scope: ConfigurationScope,
  input: unknown,
  signal?: AbortSignal,
) =>
  podRequest(
    scope,
    "inspect",
    selectionInspectionSchema,
    signal,
    { key: crypto.randomUUID(), body: selectionInputSchema.parse(input) },
    "product-selection",
  );
export const readSelectionRun = (
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
    selectionRunSchema,
    signal,
    undefined,
    "product-selection",
  );
export const prepareSelection = (
  scope: ConfigurationScope,
  intent: Extract<SelectionIntent, { kind: "prepare" }>,
  signal?: AbortSignal,
) =>
  podRequest(
    scope,
    "prepare",
    selectionPreparedSchema,
    signal,
    selectionIntentSchema.parse(intent),
    "product-selection",
  );
export function executeSelection(
  scope: ConfigurationScope,
  intent: Exclude<SelectionIntent, { kind: "prepare" }>,
  signal?: AbortSignal,
) {
  const i = selectionIntentSchema.parse(intent);
  if (i.kind === "prepare") throw new Error("Invalid execution");
  return podRequest(
    scope,
    "contexts/" +
      i.contextId +
      "/runs" +
      (i.kind === "resume" ? "/" + i.key + "/resume" : ""),
    selectionRunSchema,
    signal,
    i,
    "product-selection",
  );
}
