#!/usr/bin/env python3
"""Build the offline review page for the frozen V0.8b review package.

    python3 -I -B analysis/evidence/review_tool.py \
        --package docs/benchmark/v08b-dev/review --out docs/benchmark/v09a-dev/review/review.html

The page is one self-contained HTML file: the 98 frozen items and the
frozen form are embedded in it, it loads nothing from a network, and it
works from the file system in any current browser. A reviewer opens each
item, reads its evidence (sources, reason and span, observation, proposed
actions, validation and execution, the rule's output), chooses one of the
protocol's four labels, writes a short explanation, makes the
independence declaration, and saves form.csv and reviewer.json, in the
formats PROTOCOL.md and review_analysis.py define.

The package is read, never written: the page refuses to be built if the
items differ from the manifest's SHA-256. The page produces no label of
its own, suggests none, and contacts no model.
"""

import argparse
import csv
import hashlib
import io
import json
import os
import sys

TOOL = "dreamteam review page 1"
STATEMENT = ("I did not write, calibrate or tune these rules, nor choose the sample. "
             "I did not see the developer's own review notes.")
DECLARATIONS = [
    ("rules", "I did not write, calibrate or tune the decision-evidence rules."),
    ("sample", "I did not choose the review sample."),
    ("notes", "I did not see the developer's own review notes."),
]


def load_package(package):
    with open(os.path.join(package, "manifest.json"), encoding="utf-8") as f:
        manifest = json.load(f)
    with open(os.path.join(package, "items.json"), "rb") as f:
        raw = f.read()
    if hashlib.sha256(raw).hexdigest() != manifest["itemsSha256"]:
        raise SystemExit("the review items differ from the frozen sample")
    items = json.loads(raw.decode("utf-8"))
    with open(os.path.join(package, "form.csv"), encoding="utf-8", newline="") as f:
        text = f.read()
    rows = list(csv.reader(io.StringIO(text)))
    header, form = rows[0], [dict(zip(rows[0], r)) for r in rows[1:]]
    if [r["item"] for r in form] != [i["item"] for i in manifest["items"]]:
        raise SystemExit("the frozen form and the manifest list different items")
    if any(r["label"] or r["comment"] or r["correction"] for r in form):
        raise SystemExit("the frozen form already carries labels")
    return manifest, items, header, form


def page(manifest, items, header, form):
    data = {
        "tool": TOOL,
        "sample": manifest["seed"],
        "itemsSha256": manifest["itemsSha256"],
        "labels": manifest["labels"],
        "statement": STATEMENT,
        "declarations": [{"key": k, "text": t} for k, t in DECLARATIONS],
        "header": header,
        "form": form,
        "items": {i["item"]: i for i in items},
    }
    blob = json.dumps(data, ensure_ascii=True, sort_keys=True, separators=(",", ":")).replace("</", "<\\/")
    return TEMPLATE.replace("@@DATA@@", blob).replace("@@LOGIC@@", LOGIC).replace("@@UI@@", UI)


# The logic the page and its tests share: no DOM, no network.
LOGIC = r"""
function csvField(v) {
  var s = v == null ? "" : String(v);
  return /[",\r\n]/.test(s) ? '"' + s.replace(/"/g, '""') + '"' : s;
}
function toCSV(header, rows) {
  var out = [header.map(csvField).join(",")];
  rows.forEach(function (r) { out.push(header.map(function (h) { return csvField(r[h]); }).join(",")); });
  return out.join("\n") + "\n";
}
function parseCSV(text) {
  var rows = [], row = [], f = "", q = false, i = 0;
  text = text.replace(/^﻿/, "");
  for (; i < text.length; i++) {
    var c = text[i];
    if (q) {
      if (c === '"') { if (text[i + 1] === '"') { f += '"'; i++; } else { q = false; } }
      else { f += c; }
    } else if (c === '"') { q = true; }
    else if (c === ",") { row.push(f); f = ""; }
    else if (c === "\n" || c === "\r") {
      if (c === "\r" && text[i + 1] === "\n") { i++; }
      row.push(f); f = ""; rows.push(row); row = [];
    } else { f += c; }
  }
  if (f !== "" || row.length) { row.push(f); rows.push(row); }
  return rows.filter(function (r) { return r.length > 1 || r[0] !== ""; });
}
// The completed form: the frozen rows, with the reviewer's label,
// comment and correction, and nothing else changed.
function formRows(data, state) {
  return data.form.map(function (f) {
    var s = state[f.item] || {}, r = {};
    data.header.forEach(function (h) { r[h] = f[h]; });
    r.label = s.label || "";
    r.comment = s.comment || "";
    r.correction = s.label === "unsupported" ? (s.correction || "") : "";
    return r;
  });
}
// A saved form read back: {state, problems}. Only the frozen rows, in
// their frozen identity, are accepted.
function readForm(data, text) {
  var rows = parseCSV(text), problems = [], state = {};
  if (!rows.length || rows[0].join(",") !== data.header.join(",")) {
    return {state: {}, problems: ["the file's columns are not the review form's"]};
  }
  var want = {};
  data.form.forEach(function (f) { want[f.item] = f; });
  rows.slice(1).forEach(function (r) {
    var o = {};
    data.header.forEach(function (h, k) { o[h] = r[k] == null ? "" : r[k]; });
    var f = want[o.item];
    if (!f) { problems.push("unknown item " + o.item); return; }
    if (o.id !== f.id || o.rule !== f.rule || o.rule_output !== f.rule_output) {
      problems.push("item " + o.item + " does not match the frozen form"); return;
    }
    if (o.label && data.labels.indexOf(o.label) < 0) { problems.push("item " + o.item + ": unknown label " + o.label); return; }
    state[o.item] = {label: o.label, comment: o.comment, correction: o.correction};
  });
  return {state: state, problems: problems};
}
// What is still missing before the review is complete.
function missing(data, state, decl) {
  var out = [];
  var unlabelled = data.form.filter(function (f) { return !(state[f.item] || {}).label; }).length;
  if (unlabelled) { out.push(unlabelled + " of " + data.form.length + " items have no label"); }
  var unexplained = data.form.filter(function (f) {
    var s = state[f.item] || {};
    return s.label && s.label !== "supported" && !(s.comment || "").trim();
  }).length;
  if (unexplained) { out.push(unexplained + " labels other than supported have no explanation"); }
  if (!decl || !(decl.reviewer || "").trim()) { out.push("the reviewer's name or role is missing"); }
  data.declarations.forEach(function (d) {
    if (!decl || !decl.checks || decl.checks[d.key] !== true) { out.push("not declared: " + d.text); }
  });
  return out;
}
// reviewer.json, as PROTOCOL.md defines it, with the declarations made.
function declaration(data, decl) {
  var checks = {}, all = true;
  data.declarations.forEach(function (d) {
    checks[d.key] = !!(decl && decl.checks && decl.checks[d.key] === true);
    all = all && checks[d.key];
  });
  return {reviewer: ((decl && decl.reviewer) || "").trim(), independent: all, statement: data.statement,
          declarations: checks, sample: data.sample, itemsSha256: data.itemsSha256, tool: data.tool};
}
"""

UI = r"""
var data = JSON.parse(document.getElementById("data").textContent);
var KEY = "dreamteam-review-" + data.sample;
var state = {}, decl = {reviewer: "", checks: {}}, cur = 0;
try { var saved = JSON.parse(localStorage.getItem(KEY) || "null"); if (saved) { state = saved.state || {}; decl = saved.decl || decl; } } catch (e) {}
function save() { try { localStorage.setItem(KEY, JSON.stringify({state: state, decl: decl})); } catch (e) {} }
function el(tag, attrs, kids) {
  var e = document.createElement(tag);
  Object.keys(attrs || {}).forEach(function (k) { if (k === "text") { e.textContent = attrs[k]; } else { e.setAttribute(k, attrs[k]); } });
  (kids || []).forEach(function (c) { if (c) { e.appendChild(typeof c === "string" ? document.createTextNode(c) : c); } });
  return e;
}
function pre(v) { return el("pre", {text: typeof v === "string" ? v : JSON.stringify(v, null, 2)}); }
function field(name, node) { return el("div", {"class": "field"}, [el("h3", {text: name}), node]); }
function reasonNode(it) {
  var r = it.reason || "", s = it.reasonSpan;
  if (!r) { return el("p", {"class": "muted", text: "No reason recorded."}); }
  if (!s || s.start == null) { return el("blockquote", {text: r}); }
  return el("blockquote", {}, [r.slice(0, s.start), el("mark", {text: r.slice(s.start, s.end)}), r.slice(s.end)]);
}
function renderList() {
  var ul = document.getElementById("list"); ul.innerHTML = "";
  data.form.forEach(function (f, k) {
    var lab = (state[f.item] || {}).label;
    var li = el("li", {"class": (k === cur ? "cur " : "") + (lab ? "done" : "")}, [
      el("span", {"class": "num", text: f.item}), el("span", {"class": "lab", text: lab || "—"})]);
    li.onclick = function () { cur = k; render(); };
    ul.appendChild(li);
  });
  var n = data.form.filter(function (f) { return (state[f.item] || {}).label; }).length;
  document.getElementById("progress").textContent = n + " of " + data.form.length + " labelled";
}
function renderItem() {
  var f = data.form[cur], it = data.items[f.item], box = document.getElementById("item");
  box.innerHTML = "";
  var run = it.run || {}, rec = it.recording || {};
  box.appendChild(el("h2", {text: f.item + " · " + it.rule + " · rule output: " + (it.ruleOutput || {}).status}));
  box.appendChild(el("p", {"class": "muted", text: "Identifier " + it.id + " · decision point " + it.point + " (" + (it.time || "") + ") · model " + (run.model || "?") + " · " + (run.scenario || "") + " seed " + (run.seed || "")}));
  box.appendChild(field("Recording", pre({path: rec.path, sha256: rec.sha256, group: rec.group, replayConfig: rec.replayConfig, replayRun: run.replayRun})));
  box.appendChild(field("Sources", pre(it.sources || "not listed in this item; see the recording above")));
  box.appendChild(field("Reason (the span the rule read is marked)", reasonNode(it)));
  if (it.check) { box.appendChild(field("What the rule checked", pre(it.check))); }
  if (it.observedStaff) { box.appendChild(field("Observation: staff behind employee IDs", pre(it.observedStaff))); }
  if (it.promptExcerpt) { box.appendChild(field("Observation: prompt excerpt (what the model was shown)", pre(it.promptExcerpt))); }
  box.appendChild(field("Proposed actions", pre(it.proposed == null ? "not part of this item" : it.proposed)));
  box.appendChild(field("Validation and execution", pre(it.execution == null ? "not part of this item" : it.execution)));
  box.appendChild(field("Rule output", el("p", {}, [el("strong", {text: (it.ruleOutput || {}).status}), ". " + ((it.ruleOutput || {}).basis || "")])));
  box.appendChild(field("Evidence completeness", pre(it.completeness)));
  var s = state[f.item] || (state[f.item] = {});
  var form = el("div", {"class": "labelbox"}, [el("h3", {text: "Your label: does the evidence in this item support the rule's output?"})]);
  data.labels.forEach(function (l) {
    var id = "l-" + l, r = el("input", {type: "radio", name: "label", id: id, value: l});
    if (s.label === l) { r.checked = true; }
    r.onchange = function () { s.label = l; save(); render(); };
    form.appendChild(el("label", {"for": id, "class": "opt"}, [r, " " + l]));
  });
  var c = el("textarea", {rows: "3", placeholder: "A brief explanation (required for any label other than supported)"});
  c.value = s.comment || ""; c.oninput = function () { s.comment = c.value; save(); renderStatus(); };
  form.appendChild(el("h3", {text: "Explanation"})); form.appendChild(c);
  if (s.label === "unsupported") {
    var k = el("textarea", {rows: "2", placeholder: "What the output should have been"});
    k.value = s.correction || ""; k.oninput = function () { s.correction = k.value; save(); };
    form.appendChild(el("h3", {text: "Correction"})); form.appendChild(k);
  }
  var nav = el("div", {"class": "nav"});
  var prev = el("button", {text: "← Previous"}), next = el("button", {text: "Next →"});
  prev.disabled = cur === 0; next.disabled = cur === data.form.length - 1;
  prev.onclick = function () { cur--; render(); }; next.onclick = function () { cur++; render(); };
  nav.appendChild(prev); nav.appendChild(next); form.appendChild(nav);
  box.appendChild(form);
}
function renderStatus() {
  var m = missing(data, state, decl), st = document.getElementById("status");
  st.innerHTML = "";
  if (!m.length) { st.appendChild(el("p", {"class": "ok", text: "Complete: every item is labelled and the declaration is made."})); }
  else { st.appendChild(el("ul", {}, m.map(function (x) { return el("li", {text: x}); }))); }
}
function render() { renderList(); renderItem(); renderStatus(); }
function download(name, text, type) {
  var a = document.createElement("a");
  a.href = URL.createObjectURL(new Blob([text], {type: type}));
  a.download = name; document.body.appendChild(a); a.click(); a.remove();
}
function setup() {
  var who = document.getElementById("reviewer");
  who.value = decl.reviewer || ""; who.oninput = function () { decl.reviewer = who.value; save(); renderStatus(); };
  var box = document.getElementById("decls");
  data.declarations.forEach(function (d) {
    var cb = el("input", {type: "checkbox", id: "d-" + d.key});
    cb.checked = !!(decl.checks || {})[d.key];
    cb.onchange = function () { decl.checks = decl.checks || {}; decl.checks[d.key] = cb.checked; save(); renderStatus(); };
    box.appendChild(el("label", {"for": "d-" + d.key, "class": "opt"}, [cb, " " + d.text]));
  });
  document.getElementById("save-form").onclick = function () { download("form.csv", toCSV(data.header, formRows(data, state)), "text/csv"); };
  document.getElementById("save-decl").onclick = function () { download("reviewer.json", JSON.stringify(declaration(data, decl), null, 1) + "\n", "application/json"); };
  document.getElementById("load-form").onchange = function (ev) {
    var file = ev.target.files[0]; if (!file) { return; }
    var rd = new FileReader();
    rd.onload = function () {
      var res = readForm(data, String(rd.result));
      if (res.problems.length) { alert("Not loaded:\n" + res.problems.join("\n")); return; }
      state = res.state; save(); render();
    };
    rd.readAsText(file);
  };
  document.getElementById("sample").textContent = data.sample + " · items SHA-256 " + data.itemsSha256;
  render();
}
setup();
"""

TEMPLATE = """<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Decision Evidence Review</title>
<style>
:root { --bg: #fbfbf9; --fg: #1d1d1b; --muted: #66665f; --line: #deddd6; --panel: #ffffff; --accent: #2d5a8a; --mark: #fbe7a1; --ok: #2f6b3a; }
@media (prefers-color-scheme: dark) { :root { --bg: #171716; --fg: #e8e7e2; --muted: #a3a29a; --line: #3a3a36; --panel: #20201e; --accent: #8db4e0; --mark: #6b5a1c; --ok: #8cc79a; } }
* { box-sizing: border-box; }
body { margin: 0; background: var(--bg); color: var(--fg); font: 15px/1.5 system-ui, -apple-system, "Segoe UI", sans-serif; }
header { padding: 16px; border-bottom: 1px solid var(--line); }
header h1 { margin: 0 0 4px; font-size: 20px; }
.muted { color: var(--muted); }
main { display: grid; grid-template-columns: 220px 1fr; gap: 0; min-height: 70vh; }
nav#side { border-right: 1px solid var(--line); max-height: 80vh; overflow-y: auto; }
#list { list-style: none; margin: 0; padding: 0; }
#list li { display: flex; justify-content: space-between; padding: 4px 12px; cursor: pointer; border-bottom: 1px solid var(--line); font-variant-numeric: tabular-nums; }
#list li.cur { background: var(--panel); outline: 2px solid var(--accent); outline-offset: -2px; }
#list li .lab { color: var(--muted); font-size: 13px; }
#list li.done .lab { color: var(--ok); }
#item { padding: 16px; max-width: 980px; }
#item h2 { font-size: 17px; margin: 0 0 4px; }
.field h3, .labelbox h3 { font-size: 13px; text-transform: uppercase; letter-spacing: .04em; color: var(--muted); margin: 16px 0 4px; }
pre { background: var(--panel); border: 1px solid var(--line); padding: 8px; overflow-x: auto; white-space: pre-wrap; word-break: break-word; margin: 0; font-size: 13px; }
blockquote { background: var(--panel); border-left: 3px solid var(--accent); margin: 0; padding: 8px 12px; }
mark { background: var(--mark); color: inherit; }
.labelbox { margin-top: 20px; padding: 12px; border: 1px solid var(--line); background: var(--panel); }
.opt { display: block; padding: 2px 0; }
textarea, input[type=text] { width: 100%; font: inherit; color: var(--fg); background: var(--bg); border: 1px solid var(--line); padding: 6px; }
button { font: inherit; padding: 6px 12px; margin: 8px 8px 0 0; background: var(--panel); color: var(--fg); border: 1px solid var(--line); cursor: pointer; }
button:disabled { opacity: .4; cursor: default; }
section#finish { padding: 16px; border-top: 1px solid var(--line); max-width: 980px; }
.ok { color: var(--ok); }
@media (max-width: 720px) { main { grid-template-columns: 1fr; } nav#side { max-height: 30vh; border-right: 0; border-bottom: 1px solid var(--line); } }
</style>
</head>
<body>
<header>
<h1>Decision evidence review</h1>
<div class="muted">Sample <span id="sample"></span>. Protocol: docs/benchmark/v08b-dev/review/PROTOCOL.md. Instructions: docs/benchmark/v09a-dev/review/INSTRUCTIONS.md. This page makes no label of its own, suggests none, and uses no network.</div>
<div id="progress" class="muted"></div>
</header>
<main>
<nav id="side"><ul id="list"></ul></nav>
<div id="item"></div>
</main>
<section id="finish">
<h2>Declaration and saving</h2>
<label for="reviewer">Your name or role</label>
<input type="text" id="reviewer">
<div id="decls"></div>
<div id="status"></div>
<button id="save-form">Save form.csv</button>
<button id="save-decl">Save reviewer.json</button>
<label class="opt">Resume from a saved form.csv: <input type="file" id="load-form" accept=".csv,text/csv"></label>
</section>
<script type="application/json" id="data">@@DATA@@</script>
<script id="logic">@@LOGIC@@</script>
<script>@@UI@@</script>
</body>
</html>
"""


def main(argv=None):
    ap = argparse.ArgumentParser(description="Build the offline review page from the frozen review package.")
    ap.add_argument("--package", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    manifest, items, header, form = load_package(a.package)
    os.makedirs(os.path.dirname(os.path.abspath(a.out)), exist_ok=True)
    with open(a.out, "w", encoding="utf-8", newline="\n") as f:
        f.write(page(manifest, items, header, form))
    print(f"review page: {len(items)} items, sample {manifest['seed']}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
