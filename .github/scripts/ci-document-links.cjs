"use strict";

const fs = require("node:fs");
const path = require("node:path");
const { execFileSync } = require("node:child_process");

function isDocumentationMarkdown(filename) {
  return filename === "README.md" ||
    filename === "AGENTS.md" ||
    (filename.startsWith("docs/") && filename.endsWith(".md"));
}

// Git may append a tab to unquoted filenames with spaces, or quote names
// containing special characters. Reject unknown headers rather than skip links.
function parsePatchDestination(line) {
  const raw = line.slice("+++ ".length).split("\t", 1)[0];
  if (raw === "/dev/null") return null;
  let value = raw;

  if (raw.startsWith('"')) {
    if (!raw.endsWith('"')) throw new Error("malformed Git patch filename: " + raw);
    const bytes = [];
    const escapes = { n: 10, t: 9, r: 13, a: 7, b: 8, f: 12, v: 11, '"': 34, "\\": 92 };
    for (let i = 1; i < raw.length - 1; i++) {
      if (raw[i] !== "\\") {
        const symbol = String.fromCodePoint(raw.codePointAt(i));
        bytes.push(...Buffer.from(symbol, "utf8"));
        i += symbol.length - 1;
        continue;
      }
      const escaped = raw[++i];
      if (escaped === undefined || i >= raw.length - 1) {
        throw new Error("malformed Git patch escape: " + raw);
      }
      if (/^[0-7]$/.test(escaped)) {
        const octal = raw.slice(i, i + 3);
        if (!/^[0-7]{3}$/.test(octal)) throw new Error("malformed Git patch octal: " + raw);
        bytes.push(parseInt(octal, 8));
        i += 2;
      } else if (Object.hasOwn(escapes, escaped)) {
        bytes.push(escapes[escaped]);
      } else {
        throw new Error("unsupported Git patch escape: " + raw);
      }
    }
    value = new TextDecoder("utf-8", { fatal: true }).decode(Uint8Array.from(bytes));
  }

  if (!value.startsWith("b/")) throw new Error("unexpected Git patch destination: " + value);
  return value.slice(2);
}

function addedMarkdownLinks(patch) {
  const links = [];
  let filename = null;
  for (const line of patch.split(/\r?\n/)) {
    if (line.startsWith("diff --git ")) filename = null;
    if (line.startsWith("+++ ")) {
      filename = parsePatchDestination(line);
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
    if ((target !== root && !target.startsWith(root + path.sep)) || !exists(target)) {
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

module.exports = { isDocumentationMarkdown, parsePatchDestination, addedMarkdownLinks, checkAddedLinks };
