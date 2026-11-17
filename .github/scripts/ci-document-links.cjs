"use strict";

const fs = require("node:fs");
const path = require("node:path");
const { execFileSync } = require("node:child_process");

function isDocumentationMarkdown(filename) {
  return filename === "README.md" ||
    filename === "AGENTS.md" ||
    (filename.startsWith("docs/") && filename.endsWith(".md"));
}

function addedMarkdownLinks(patch) {
  const links = [];
  let filename = null;
  for (const line of patch.split(/\r?\n/)) {
    if (line.startsWith("diff --git ")) filename = null;
    if (line.startsWith("+++ b/")) {
      filename = line.slice("+++ b/".length);
      continue;
    }
    if (line.startsWith("+++ /dev/null")) {
      filename = null;
      continue;
    }
    if (!filename || !isDocumentationMarkdown(filename) ||
        !line.startsWith("+") || line.startsWith("+++ ")) continue;
    const addedLine = line.slice(1);
    // Validate added inline links only, not historic content or remote URLs.
    for (const match of addedLine.matchAll(/!?\[[^\]]+\]\((<[^>]+>|[^\s)]+)(?:\s+["'][^"']+["'])?\)/g)) {
      links.push({ filename, href: match[1].replace(/^<|>$/g, "") });
    }
  }
  return links;
}

function checkAddedLinks(patch, { repositoryRoot = process.cwd(), exists = fs.existsSync } = {}) {
  const errors = [];
  const root = path.resolve(repositoryRoot);
  for (const { filename, href } of addedMarkdownLinks(patch)) {
    if (href.startsWith("#") || href.startsWith("/") ||
        /^[a-z][a-z0-9+.-]*:/i.test(href)) continue;
    const reference = href.split(/[?#]/, 1)[0];
    if (!reference) continue;
    let decoded;
    try {
      decoded = decodeURIComponent(reference);
    } catch {
      errors.push(filename + ": invalid URL encoding in " + href);
      continue;
    }
    const target = path.resolve(root, path.dirname(filename), decoded);
    if (!target.startsWith(root + path.sep) || !exists(target)) {
      errors.push(filename + ": unresolved relative link " + href);
    }
  }
  return errors;
}

function main(argv) {
  if (argv.length !== 2 || !argv.every((sha) => /^[0-9a-f]{40}$/.test(sha))) {
    throw new Error("usage: node ci-document-links.cjs <base-sha> <head-sha>");
  }
  const [base, head] = argv;
  const patch = execFileSync("git", [
    "diff", "--no-renames", "--unified=0", base + "..." + head,
    "--", "AGENTS.md", "README.md", "docs/",
  ], { encoding: "utf8", maxBuffer: 16 * 1024 * 1024 });
  const checked = addedMarkdownLinks(patch);
  const errors = checkAddedLinks(patch);
  if (errors.length > 0) throw new Error(errors.join("\n"));
  process.stdout.write("Verified " + checked.length + " added Markdown links against current files\n");
}

if (require.main === module) {
  try {
    main(process.argv.slice(2));
  } catch (error) {
    process.stderr.write(error.message + "\n");
    process.exitCode = 1;
  }
}

module.exports = { isDocumentationMarkdown, addedMarkdownLinks, checkAddedLinks };
