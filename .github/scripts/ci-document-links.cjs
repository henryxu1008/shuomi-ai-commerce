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

function referenceKey(label) {
  return label.trim().replace(/\s+/g, " ").toLowerCase();
}

function unescapeLinkPath(href) {
  return href.replace(/\\([()[\]<>\\ ])/g, "$1");
}

function parseReferenceDefinition(line) {
  const match = line.match(/^ {0,3}\[([^\]]+)\]:[ \t]*(?:<([^>]+)>|(\S+))/);
  if (!match) return null;
  return {
    kind: "definition",
    key: referenceKey(match[1]),
    href: unescapeLinkPath(match[2] ?? match[3]),
    column: line.indexOf("["),
  };
}

function parseParenthesizedDestination(line, opening) {
  let depth = 1;
  let closing = -1;
  for (let at = opening + 1; at < line.length; at++) {
    if (line[at] === "\\") {
      at++;
      continue;
    }
    if (line[at] === "(") depth++;
    else if (line[at] === ")" && --depth === 0) {
      closing = at;
      break;
    }
  }
  if (closing === -1) return null;
  let href = line.slice(opening + 1, closing).trim();
  if (href.startsWith("<")) {
    const endAngle = href.indexOf(">");
    if (endAngle === -1) return null;
    href = href.slice(1, endAngle);
  } else {
    // Optional Markdown title after the destination is not part of the URL.
    const quotedTitle = href.match(/^(\S+)\s+(["'])(?:[\s\S]*)\2$/);
    const parenthesizedTitle = href.match(/^(\S+)\s+\((?:[\s\S]*)\)$/);
    if (quotedTitle || parenthesizedTitle) {
      href = (quotedTitle || parenthesizedTitle)[1];
    }
  }
  return { href: unescapeLinkPath(href), end: closing + 1 };
}

function scanAddedMarkdownLinks(line) {
  const definition = parseReferenceDefinition(line);
  if (definition) return [definition];
  const links = [];
  for (let i = 0; i < line.length;) {
    if (line[i] !== "[" || (i > 0 && line[i - 1] === "\\")) {
      i++;
      continue;
    }
    let nesting = 1;
    let closing = -1;
    for (let j = i + 1; j < line.length; j++) {
      if (line[j] === "\\") {
        j++;
        continue;
      }
      if (line[j] === "[") nesting++;
      else if (line[j] === "]" && --nesting === 0) {
        closing = j;
        break;
      }
    }
    if (closing === -1) {
      i++;
      continue;
    }
    const label = line.slice(i + 1, closing);
    const column = i > 0 && line[i - 1] === "!" ? i - 1 : i;
    const after = closing + 1;
    if (line[after] === "(") {
      const target = parseParenthesizedDestination(line, after);
      if (target) {
        links.push({ kind: "inline", href: target.href, column });
        i = target.end;
        continue;
      }
    } else if (line[after] === "[") {
      const endRef = line.indexOf("]", after + 1);
      if (endRef !== -1) {
        const reference = line.slice(after + 1, endRef) || label;
        links.push({ kind: "reference", key: referenceKey(reference), href: null, column });
        i = endRef + 1;
        continue;
      }
    }
    links.push({ kind: "shortcut", key: referenceKey(label), href: null, column });
    i = closing + 1;
  }
  return links;
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
        for (const token of scanAddedMarkdownLinks(addedLine)) {
          const link = { filename, href: token.href };
          Object.defineProperties(link, {
            lineNumber: { value: newLine },
            column: { value: token.column },
            kind: { value: token.kind },
            referenceKey: { value: token.key ?? null },
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
// Only a matched pair of Markdown backtick runs denotes a code span.
// Unmatched delimiters are literal text and may not conceal real links.
function maskInlineCode(content) {
  const chars = content.split("");
  const isEscaped = (at) => {
    let count = 0;
    for (let p = at - 1; p >= 0 && content[p] === "\\"; p--) count++;
    return count % 2 === 1;
  };
  for (let i = 0; i < content.length;) {
    if (content[i] !== "\x60" || isEscaped(i)) {
      i++;
      continue;
    }
    let size = 1;
    while (content[i + size] === "\x60") size++;
    let close = -1;
    for (let j = i + size; j < content.length;) {
      if (content[j] !== "\x60" || isEscaped(j)) {
        j++;
        continue;
      }
      let length = 1;
      while (content[j + length] === "\x60") length++;
      if (length === size) {
        close = j;
        break;
      }
      j += length;
    }
    if (close === -1) {
      i += size;
      continue;
    }
    for (let j = i; j < close + size; j++) {
      if (chars[j] !== "\n") chars[j] = " ";
    }
    i = close + size;
  }
  return chars.join("");
}

function maskMarkdownCode(content) {
  const lines = content.split(/\r?\n/);
  const masked = Array(lines.length);
  let fence = null;
  let plainStart = 0;
  const flushPlain = (end) => {
    if (end <= plainStart) return;
    const visible = maskInlineCode(lines.slice(plainStart, end).join("\n")).split("\n");
    visible.forEach((line, index) => { masked[plainStart + index] = line; });
  };
  for (let index = 0; index < lines.length; index++) {
    const line = lines[index];
    const delimiter = line.match(/^ {0,3}(\x60{3,}|~{3,})(.*)$/);
    let openedHere = false;
    if (!fence && delimiter) {
      flushPlain(index);
      fence = { char: delimiter[1][0], length: delimiter[1].length };
      openedHere = true;
    }
    if (!fence) continue;
    masked[index] = " ".repeat(line.length);
    if (!openedHere && delimiter && delimiter[1][0] === fence.char &&
        delimiter[1].length >= fence.length && delimiter[2].trim() === "") {
      fence = null;
    }
    plainStart = index + 1;
  }
  flushPlain(lines.length);
  return masked;
}

function checkAddedLinks(patch, {
  repositoryRoot = process.cwd(),
  exists = fs.existsSync,
  readFile = fs.readFileSync,
} = {}) {
  const errors = [];
  const root = path.resolve(repositoryRoot);
  const documents = new Map();

  for (const link of addedMarkdownLinks(patch)) {
    const { filename } = link;
    if (!documents.has(filename)) {
      const source = readFile(path.resolve(root, filename), "utf8");
      const visible = maskMarkdownCode(source);
      const definitions = new Map();
      for (const line of visible) {
        const definition = parseReferenceDefinition(line);
        if (definition && !definitions.has(definition.key)) {
          definitions.set(definition.key, definition.href);
        }
      }
      documents.set(filename, { visible, definitions });
    }
    const document = documents.get(filename);
    const visibleLine = document.visible[link.lineNumber - 1];
    if (visibleLine === undefined) {
      throw new Error(filename + ": diff line is outside the current document");
    }
    // Links in code spans or fences are literal examples, not rendered links.
    if (visibleLine[link.column] !== "[" && visibleLine[link.column] !== "!") continue;

    let href = link.href;
    if (link.kind === "reference" || link.kind === "shortcut") {
      href = document.definitions.get(link.referenceKey);
      if (href === undefined) {
        if (link.kind === "reference") {
          errors.push(filename + ": missing Markdown reference " + link.referenceKey);
        }
        continue;
      }
    }
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
  // Hunk line coordinates refer to PR HEAD, not the synthetic merge checkout.
  const errors = checkAddedLinks(patch, {
    readFile: (absoluteFilename) => {
      // checkAddedLinks uses filesystem paths; Git rev:path requires a
      // repository-relative tree path, never an absolute checkout path.
      const relative = path.relative(process.cwd(), absoluteFilename);
      if (!relative || relative === ".." || relative.startsWith(".." + path.sep) ||
          path.isAbsolute(relative)) {
        throw new Error("document path escapes repository: " + absoluteFilename);
      }
      return execFileSync("git", [
        "show", head + ":" + relative.split(path.sep).join("/"),
      ], { encoding: "utf8", maxBuffer: 16 * 1024 * 1024 });
    },
  });
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
