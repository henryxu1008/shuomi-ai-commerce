"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { isDocumentationMarkdown, addedMarkdownLinks, checkAddedLinks } = require("./ci-document-links.cjs");

test("only repository Markdown paths are treated as documentation link inputs", () => {
  assert.equal(isDocumentationMarkdown("docs/product/README.md"), true);
  assert.equal(isDocumentationMarkdown("README.md"), true);
  assert.equal(isDocumentationMarkdown("AGENTS.md"), true);
  assert.equal(isDocumentationMarkdown("docs/document-inventory.csv"), false);
  assert.equal(isDocumentationMarkdown("internal/pkg/doc.go"), false);
});

test("new relative documentation references are checked but old and removed lines are ignored", () => {
  const patch = [
    "diff --git a/docs/README.md b/docs/README.md",
    "--- a/docs/README.md",
    "+++ b/docs/README.md",
    "@@ -1,2 +1,3 @@",
    "-[removed](missing-old.md)",
    " [unchanged](missing-unchanged.md)",
    "+[valid](architecture/README.md#section) and [missing](architecture/missing.md)",
    "+[web](https://example.com), [anchor](#section), [email](mailto:foo@example.com)",
    "diff --git a/docs/document-inventory.csv b/docs/document-inventory.csv",
    "--- /dev/null",
    "+++ b/docs/document-inventory.csv",
    "@@ -0,0 +1,1 @@",
    "+path,layer,domain",
  ].join("\n");
  const links = addedMarkdownLinks(patch);
  assert.deepEqual(links.map((x) => x.href), [
    "architecture/README.md#section",
    "architecture/missing.md",
    "https://example.com",
    "#section",
    "mailto:foo@example.com",
  ]);
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "doc-ci-"));
  try {
    fs.mkdirSync(path.join(root, "docs", "architecture"), { recursive: true });
    fs.writeFileSync(path.join(root, "docs", "architecture", "README.md"), "ok");
    assert.deepEqual(checkAddedLinks(patch, { repositoryRoot: root }), [
      "docs/README.md: unresolved relative link architecture/missing.md",
    ]);
    fs.writeFileSync(path.join(root, "docs", "architecture", "missing.md"), "ok");
    assert.deepEqual(checkAddedLinks(patch, { repositoryRoot: root }), []);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("unrecognized and malformed relative targets fail closed, while valid encoded names work", () => {
  const patch = [
    "diff --git a/docs/README.md b/docs/README.md",
    "+++ b/docs/README.md",
    "+[out](../../outside.md), [bad](bad%ZZ.md), [encoded](file%20name.md)",
  ].join("\n");
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "doc-ci-"));
  try {
    fs.mkdirSync(path.join(root, "docs"), { recursive: true });
    fs.writeFileSync(path.join(root, "docs", "file name.md"), "ok");
    assert.deepEqual(checkAddedLinks(patch, { repositoryRoot: root }), [
      "docs/README.md: unresolved relative link ../../outside.md",
      "docs/README.md: invalid URL encoding in bad%ZZ.md",
    ]);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("new document with directory links is validated without checking removed content", () => {
  const patch = [
    "diff --git a/docs/product/flows/README.md b/docs/product/flows/README.md",
    "--- /dev/null",
    "+++ b/docs/product/flows/README.md",
    "@@ -0,0 +1,2 @@",
    "+# Flows",
    "+[platform](../../../internal/marketplace/) and [products](../README.md)",
  ].join("\n");
  assert.deepEqual(checkAddedLinks(patch, {
    repositoryRoot: "/tmp/example-repo",
    exists: (p) => p === "/tmp/example-repo/internal/marketplace" ||
      p === "/tmp/example-repo/docs/product/README.md",
  }), []);
});
