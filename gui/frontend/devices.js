// The Devices tab: one row per physical controller, with the templates ranked
// against the bindings that controller actually uses.
import { api, el, setStatus, postJSON } from "./core.js";
import { openPreview } from "./diagrams.js";
import { t, tMessage } from "./i18n.js";

export const devices = { payload: null, expanded: null };

function badge(text, className) {
  return el("span", `badge ${className || ""}`.trim(), text);
}

// templateRow is one candidate in the picker, with its fit against this device.
function templateRow(entry, option) {
  const row = el("div", "template-row");
  if (option.path === entry.templatePath) row.classList.add("is-current");

  const main = el("div", "template-main");
  main.append(el("span", "template-name", option.name));

  const keys = el("span", "template-keys");
  keys.append(
    badge(t("devices.buttonsBadge", { n: option.buttons })),
    badge(t("devices.axesBadge", { n: option.axes })),
    badge(t("devices.hatsBadge", { n: option.hats })),
  );
  main.append(keys);
  row.append(main);

  // The fit bar is the point of this list: how much of what the device
  // actually uses this template can display.
  const fit = el("div", "template-fit");
  const bar = el("div", "fit-bar");
  const fill = el("div", "fit-fill");
  fill.style.width = option.total ? `${Math.round((option.score / option.total) * 100)}%` : "0%";
  fill.classList.add(option.compatible ? "is-good" : "is-poor");
  bar.append(fill);
  fit.append(bar);

  const label = t("devices.keysUsed", { score: option.score, total: option.total });
  const missing = option.missing && option.missing.length
    ? t("devices.missingInputs", { n: option.missing.length })
    : "";
  fit.append(el("span", "template-fit-label", label + missing));
  row.append(fit);

  const actions = el("div", "template-actions");

  const preview = el("button", "btn btn-small", t("devices.preview"));
  preview.type = "button";
  preview.addEventListener("click", () => openPreview(option.path, option.name));
  actions.append(preview);

  const assign = el("button", "btn btn-small btn-primary", t("devices.assign"));
  assign.type = "button";
  assign.addEventListener("click", () =>
    postDeviceMapping({
      action: "assign",
      guid: entry.guid,
      name: entry.name,
      templatePath: option.path,
    }),
  );
  actions.append(assign);

  row.append(actions);
  return row;
}

function templatePicker(entry) {
  const picker = el("div", "template-picker");

  if (!entry.templates.length) {
    picker.append(el("p", "empty-note", t("devices.noTemplateFound")));
    return picker;
  }

  picker.append(el("p", "picker-note", t("devices.ranking")));
  for (const option of entry.templates) {
    picker.append(templateRow(entry, option));
  }

  // TARGET device number, only meaningful for Thrustmaster setups.
  const target = el("div", "target-row");
  target.append(el("label", "target-label", t("devices.targetNumber")));
  const input = el("input", "field-input target-input");
  input.type = "number";
  input.value = entry.targetNumber || "";
  input.placeholder = "1001";
  target.append(input);
  const apply = el("button", "btn btn-small", t("devices.set"));
  apply.type = "button";
  apply.addEventListener("click", () =>
    postJSON("/api/devices/target", {
      guid: entry.guid,
      name: entry.name,
      targetNumber: Number(input.value) || 0,
    }).then(renderDevices),
  );
  target.append(apply);
  picker.append(target);

  return picker;
}

function deviceRow(entry) {
  const row = el("div", "device-row");
  if (entry.isVirtual) row.classList.add("is-virtual");

  const head = el("div", "device-head");

  const identity = el("div", "device-identity");
  identity.append(el("span", "device-name", entry.name));

  const badges = el("div", "device-badges");
  for (const sim of entry.simulators) badges.append(badge(sim));
  if (entry.isVirtual) badges.append(badge(t("devices.virtual"), "badge-muted"));
  badges.append(badge(t("devices.bindings", { n: entry.bindings }), "badge-muted"));
  identity.append(badges);
  head.append(identity);

  const assignment = el("div", "device-assignment");
  if (entry.skipped) {
    assignment.append(el("span", "assignment-skipped", t("devices.ignored")));
  } else if (entry.templateName) {
    assignment.append(el("span", "assignment-template", entry.templateName));
  } else {
    assignment.append(el("span", "assignment-none", t("devices.noTemplate")));
  }
  head.append(assignment);

  const actions = el("div", "device-actions");

  if (entry.templatePath && !entry.skipped) {
    const preview = el("button", "btn btn-small", t("devices.preview"));
    preview.type = "button";
    preview.addEventListener("click", () => openPreview(entry.templatePath, entry.templateName));
    actions.append(preview);
  }

  const choose = el("button", "btn btn-small",
    entry.templateName ? t("devices.change") : t("devices.assign"));
  choose.type = "button";
  choose.addEventListener("click", () => {
    devices.expanded = devices.expanded === entry.guid ? null : entry.guid;
    renderDevices(devices.payload);
  });
  actions.append(choose);

  if (!entry.skipped) {
    const ignore = el("button", "btn btn-small", t("devices.ignore"));
    ignore.type = "button";
    ignore.addEventListener("click", () =>
      postDeviceMapping({ action: "skip", guid: entry.guid, name: entry.name }),
    );
    actions.append(ignore);
  }

  head.append(actions);
  row.append(head);

  if (devices.expanded === entry.guid) row.append(templatePicker(entry));

  return row;
}

export function renderDevices(payload) {
  devices.payload = payload;

  const body = document.getElementById("devices-body");
  body.replaceChildren();

  for (const warning of payload.warnings || []) {
    body.append(el("p", "warning-banner", tMessage(warning)));
  }

  if (!payload.devices.length) {
    body.append(el("p", "empty-note",
      t("devices.none")));
    document.getElementById("devices-summary").textContent = "";
    return;
  }

  const list = el("div", "device-list");
  for (const entry of payload.devices) list.append(deviceRow(entry));
  body.append(list);

  const assigned = payload.devices.filter((d) => d.templatePath && !d.skipped).length;
  document.getElementById("devices-summary").textContent = t("devices.summary", {
    n: payload.devices.length,
    assigned,
    total: payload.templateCount,
  });
}

// detectTargetNumbers fills in the Thrustmaster device numbers from the TARGET
// profile, matching by controller name. Nobody should have to know that a
// Warthog stick is device 1001.
export async function detectTargetNumbers() {
  const button = document.getElementById("btn-detect-target");
  button.disabled = true;

  try {
    const result = await postJSON("/api/devices/target/detect", {});
    devices.expanded = null;
    renderDevices(result.devices);
    if (result.matched) setStatus("msg.targetDetected", { n: result.matched });
    else setStatus("msg.targetDetectedNone");
  } catch (err) {
    setStatus("msg.targetDetectFailed", { error: err.message }, true);
  } finally {
    button.disabled = false;
  }
}

async function postDeviceMapping(body) {
  try {
    devices.expanded = null;
    renderDevices(await postJSON("/api/devices/mapping", body));
    setStatus("msg.configSaved");
  } catch (err) {
    setStatus("msg.deviceUpdateFailed", { error: err.message }, true);
  }
}

export async function scanDevices() {
  const button = document.getElementById("btn-scan");
  button.disabled = true;
  setStatus("msg.scanning");
  try {
    renderDevices(await api("/api/devices"));
    setStatus("msg.scanComplete");
  } catch (err) {
    setStatus("msg.scanFailed", { error: err.message }, true);
  } finally {
    button.disabled = false;
  }
}
