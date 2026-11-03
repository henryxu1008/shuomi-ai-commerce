import { expect, it } from "vitest";
import { researchLocation, researchNavigationHref, researchStep } from "./research-navigation";

const root = "/workbench/data/research", project = "11111111-1111-4111-8111-111111111111";
const choices = [project, "22222222-2222-4222-8222-222222222222"].map(id => ({
  candidateId: id, expectedRevision: "2", linkId: id, evidenceDigest: "a".repeat(64),
  selection: { itemId: id, expectedRevision: "1", originalPublicationId: id, originalVersion: "1" },
}));

it("keeps exact selected observations only inside the same project workflow", () => {
  const sample = JSON.stringify(choices), query = new URLSearchParams({step:"evidence",sample,candidateAfter:project,mode:"archived",search:"收纳",ignored:"discard"});
  const location = researchLocation(`${root}/${project}`, query)!;
  const target = new URL(researchNavigationHref(`${root}?step=summary`, location), "https://test.invalid");
  expect(target.pathname).toBe(location.pathname);
  expect(target.searchParams.get("sample")).toBe(sample);
  expect(target.searchParams.get("candidateAfter")).toBe(project);
  expect(target.searchParams.has("ignored")).toBe(false);
  expect(researchNavigationHref("/workbench/supply/selection?step=analysis", location)).toBe("/workbench/supply/selection?step=analysis");
  const list = new URL(researchNavigationHref(`${root}?step=summary`, researchLocation(root,query)), "https://test.invalid");
  expect(list.searchParams.has("sample")).toBe(false);expect(list.searchParams.has("candidateAfter")).toBe(false);
});

it("ignores invalid, repeated or foreign steps and malformed selection context", () => {
  expect(researchStep("research","analysis")).toBeUndefined();
  expect(researchLocation(root,new URLSearchParams("step=evidence&step=summary"))?.step).toBeUndefined();
  expect(researchLocation(`${root}/not-a-project`,new URLSearchParams())).toBeUndefined();
  const location = researchLocation(`${root}/${project}`,new URLSearchParams("sample=%7Bbroken&candidateAfter=bad&mode=bad"));
  expect(researchNavigationHref(`${root}?step=summary`,location)).toBe(`${root}/${project}?step=summary`);
});
