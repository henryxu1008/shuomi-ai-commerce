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
  let newLine = 1;
  for (const line of patch.split(/\r?\n/)) {
    if (line.startsWith("diff --git ")) {
      filename = null;
      newLine = 1;
      continue;
    }
    if (line.startsWith("+++ ")) {
      filename = parsePatchDestination(line);
      continue;
    }
    if (line.startsWith("@@ ")) {
      const match = line.match(/^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@/);
      if (!match) throw new Error("unsupported Git diff hunk header: " + line);
      newLine = Number(match[1]);
      continue;
    }
    if (line.startsWith("+") && !line.startsWith("+++ ")) {
      if (filename && isDocumentationMarkdown(filename)) {
        const addedLine = line.slice(1);
        for (const match of addedLine.matchAll(/!?\[[^\]]+\]\((<[^>]+>|[^\s)]+)(?:\s+["'][^"']+["'])?\)/g)) {
          const link = { filename, href: match[1].replace(/^<|>$/g, "") };
          // Preserve the existing public shape; metadata is only used by checkAddedLinks.
          Object.defineProperties(link, {
            lineNumber: { value: newLine },
            column: { value: match.index },
          });
          links.push(link);
        }
      }
      newLine++;
    } else if (line.startsWith(" ")) {
      newLine++;
    }
  }
  return links;
}

// Read the complete current document so links added *inside an existing* code
// fence are also ignored, even when the opening fence was not part of the diff.
function maskMarkdownCode(content) {
  const masked = [];
  let fence = null;
  let inlineTicks = 0;
  for (const line of content.split(/\r?\n/)) {
    const delimiter = line.match(/^ {0,3}(`{3,}|~{3,})(.*)$/);
    if (delimiter && !inlineTicks) {
      const run = delimiter[1];
      if (!fence) {
        fence = { char: run[0], length: run.length };
        masked.push(" ".repeat(line.length));
        continue;
      }
      if (run[0] === fence.char && run.length >= fence.length &&
          delimiter[2].trim() === "") {
        fence = null;
        masked.push(" ".repeat(line.length));
        continue;
      }
    }
    if (fence) {
      masked.push(" ".repeat(line.length));
      continue;
    }

    let visible = "";
    for (let i = 0; i < line.length;) {
      if (line[i] === "\x60" && (i === 0 || line[i - 1] !== "\\")) {
        let run = 1;
        while (line[i + run] === "\x60") run++;
        if (inlineTicks === 0) inlineTicks = run;
        else if (inlineTicks === run) inlineTicks = 0;
        visible += " ".repeat(run);
        i += run;
      } else {
        visible += inlineTicks ? " " : line[i];
        i++;
      }
    }
    masked.push(visible);
  }
  return masked;
}

function checkAddedLinks(patch, {
  repositoryRoot = process.cwd(),
  exists = fs.existsSync,
  readFile = fs.readFileSync,
} = {}) {
  const errors = [];
  const root = path.resolve(repositoryRoot);
  const codeMasks = new Map();

  for (const link of addedMarkdownLinks(patch)) {
    const { filename, href } = link;
    if (!codeMasks.has(filename)) {
      const source = readFile(path.resolve(root, filename), "utf8");
      codeMasks.set(filename, maskMarkdownCode(source));
    }
    const visibleLine = codeMasks.get(filename)[link.lineNumber - 1];
    if (visibleLine === undefined) {
      throw new Error(filename + ": diff line is outside the current document");
    }
    // A regex match inside a fenced block or inline code is a literal example.
    if (visibleLine[link.column] !== "[" && visibleLine[link.column] !== "!") continue;

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
