// The Generate tab: starting an export, following it, and reporting what came
// out of it.
import { api, el, setStatus, postJSON, opened } from "./core.js";
import { t, tMessage } from "./i18n.js";

export const exportTargets = { payload: null };

export async function loadExportTargets() {
  try {
    renderExportTargets(await api("/api/export/targets"));
  } catch (err) {
    setStatus("msg.targetsFailed", { error: err.message }, true);
  }
}

export function renderExportTargets(payload) {
  exportTargets.payload = payload;

  const select = document.getElementById("export-target");
  const previous = select.value;
  select.replaceChildren();
  for (const target of payload.targets) {
    // Simulator and module names are proper nouns; only the "everything" entry
    // is a phrase, and it arrives as a code.
    const option = el("option", null, target.labelCode ? t(target.labelCode) : target.label);
    option.value = target.filter;
    select.append(option);
  }
  select.value = previous;

  const warnings = document.getElementById("export-warnings");
  warnings.replaceChildren();
  for (const warning of payload.warnings || []) {
    warnings.append(el("p", "warning-banner", tMessage(warning)));
  }
  // draw.io missing only costs the PNG copies; the SVG diagrams are still
  // produced, so this is information, not a blocker.
  if (payload.drawio && payload.drawio.severity === "warn") {
    warnings.append(el("p", "warning-banner", tMessage(payload.drawio.detail)));
  }
}

function appendLog(line) {
  const log = document.getElementById("export-log");
  const atBottom = log.scrollHeight - log.scrollTop - log.clientHeight < 40;
  log.append(line + "\n");
  // Follow the output unless the user scrolled up to read something.
  if (atBottom) log.scrollTop = log.scrollHeight;
}

// Validation errors are not failures. The diagram is produced and that one
// binding just has nowhere to go, so they get their own list instead of scrolling past
// in the log.
function renderValidationErrors(errors) {
  if (!errors || !errors.length) return null;

  const box = el("details", "validation-box");
  box.append(el("summary", null,
    t("generate.validationTitle", { n: errors.length })));

  const table = el("table", "validation-table");
  const head = el("tr");
  const columns = [
    t("generate.colDevice"),
    t("generate.colSimulator"),
    t("generate.colInput"),
    t("generate.colAction"),
    t("generate.colMissingKey"),
  ];
  for (const label of columns) {
    head.append(el("th", null, label));
  }
  table.append(head);

  for (const err of errors) {
    const row = el("tr");
    row.append(
      el("td", null, err.deviceName),
      el("td", null, err.simulator),
      el("td", null, `${err.inputType} ${err.inputId}`),
      el("td", null, err.action),
      el("td", "mono", err.missingKey),
    );
    table.append(row);
  }

  box.append(table);
  return box;
}

// lastSummary keeps the finished export's outcome so a change of language can
// redraw it. The log above it cannot follow: it is the pipeline's own output,
// which stays in English (it is the same text simdiag.exe -b prints).
export let lastSummary = null;

export function renderSummary(summary, cancelled) {
  lastSummary = summary ? { summary, cancelled } : null;

  const node = document.getElementById("export-summary");
  node.replaceChildren();
  if (!summary) return;

  const box = el("div", "summary-box");
  if (cancelled) box.classList.add("is-cancelled");

  const stats = el("div", "summary-stats");
  const stat = (value, label) => {
    const cell = el("div", "summary-stat");
    cell.append(el("span", "summary-value", String(value)), el("span", "summary-label", label));
    return cell;
  };
  stats.append(
    stat(summary.devices, t("generate.statDevices")),
    stat(summary.bindings, t("generate.statBindings")),
    stat(`${(summary.durationMs / 1000).toFixed(1)}s`, t("generate.statElapsed")),
  );
  box.append(stats);

  for (const sim of summary.simulators || []) {
    const line = t("generate.simulatorLine", {
      simulator: sim.simulator,
      devices: sim.devices,
      bindings: sim.bindings,
    }) + (sim.modules && sim.modules.length ? ` — ${sim.modules.join(", ")}` : "");
    box.append(el("p", "summary-line", line));
  }

  if (summary.csvPath) {
    box.append(el("p", "summary-line mono", summary.csvPath));
  }

  for (const error of summary.errors || []) {
    box.append(el("p", "summary-error", error));
  }
  for (const warning of summary.warnings || []) {
    box.append(el("p", "summary-warning", warning));
  }

  const validation = renderValidationErrors(summary.validationErrors);
  if (validation) box.append(validation);

  node.append(box);
}

export function setExporting(running) {
  document.getElementById("btn-export").disabled = running;
  document.getElementById("btn-cancel").hidden = !running;
  document.getElementById("export-target").disabled = running;
  document.getElementById("export-csv-only").disabled = running;
}

// The run is started, then polled. A long-lived streaming response would be
// simpler, but Wails' asset server buffers a handler's response until it
// returns, so nothing would reach the page until the export was already over.
export async function runExport() {
  document.getElementById("export-log").textContent = "";
  document.getElementById("export-summary").replaceChildren();
  setExporting(true);
  setStatus("msg.exportRunning");

  try {
    const initial = await api("/api/export/start", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        filter: document.getElementById("export-target").value,
        noSvg: document.getElementById("export-csv-only").checked,
      }),
    });

    await pollExport(initial.nextIndex || 0);
  } catch (err) {
    setStatus("msg.exportFailed", { error: err.message }, true);
    setExporting(false);
  }
}

export async function pollExport(from) {
  for (;;) {
    const state = await api(`/api/export/state?from=${from}`);

    for (const line of state.lines) appendLog(line);
    from = state.nextIndex;

    if (!state.running) {
      setExporting(false);
      renderSummary(state.summary, state.cancelled);

      if (state.cancelled) setStatus("msg.exportCancelled");
      else if (state.error) setStatus("msg.exportFailed", { error: state.error }, true);
      else setStatus("msg.exportComplete");

      // The diagrams on disk changed, so anything already shown is stale.
      opened.delete("diagrams");
      return;
    }

    await new Promise((resolve) => setTimeout(resolve, 250));
  }
}

export async function cancelExport() {
  document.getElementById("btn-cancel").disabled = true;
  try {
    await postJSON("/api/export/cancel", {});
    setStatus("msg.cancelling");
  } catch (err) {
    setStatus("msg.cancelFailed", { error: err.message }, true);
  } finally {
    document.getElementById("btn-cancel").disabled = false;
  }
}
