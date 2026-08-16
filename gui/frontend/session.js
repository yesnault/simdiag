// Which configuration is open, and which language the interface speaks.
//
// It imports every tab because changeLanguage re-renders all of them from the
// payloads they already hold, and resetAfterSwitch drops those payloads when the
// configuration changes. That fan-out is the dependency itself, now visible.
import { api, el, setStatus, renderStatus, postJSON, opened, onFirstOpen, askConfirm } from "./core.js";
import { t, currentLanguage, setCurrentLanguage } from "./i18n.js";
import { config, collectConfig, renderConfig, markDirty, loadConfig } from "./config.js";
import { devices, renderDevices } from "./devices.js";
import { diagrams, renderDiagrams, closePreview } from "./diagrams.js";
import { exportTargets, renderExportTargets, lastSummary, renderSummary } from "./generate.js";
import { renderAbout } from "./about.js";

export const session = { payload: null };

export async function loadSession() {
  try {
    applySession(await api("/api/session"));
  } catch (err) {
    setStatus("msg.backendUnreachable", { error: err.message }, true);
  }
}

export function applySession(payload) {
  session.payload = payload;
  // No stored preference yet: follow the browser, which reports the language
  // Windows is running in.
  setCurrentLanguage(payload.language || navigator.language);
  applyStaticTranslations();
  markLanguageButtons();

  document.getElementById("version").textContent = payload.version;
  document.getElementById("about-version").textContent = payload.version;
  document.getElementById("config-path").textContent = payload.configPath;
  document.getElementById("config-current").title = payload.configPath;
  renderRecentMenu(payload.recent || []);
}

// ---------------------------------------------------------------- language

// applyStaticTranslations rewrites the markup that index.html spells out, which
// is everything the panels do not build themselves.
function applyStaticTranslations() {
  for (const node of document.querySelectorAll("[data-i18n]")) {
    node.textContent = t(node.dataset.i18n);
  }
  for (const node of document.querySelectorAll("[data-i18n-title]")) {
    node.title = t(node.dataset.i18nTitle);
  }
  for (const node of document.querySelectorAll("[data-i18n-placeholder]")) {
    node.placeholder = t(node.dataset.i18nPlaceholder);
  }
}

function markLanguageButtons() {
  for (const button of document.querySelectorAll(".lang")) {
    button.setAttribute("aria-pressed", String(button.dataset.language === currentLanguage));
  }
}

// changeLanguage redraws the interface from what is already loaded. Nothing is
// fetched again: every panel keeps the payload it rendered from, and Go sends
// message codes rather than sentences precisely so this is possible.
export async function changeLanguage(lang) {
  if (lang === currentLanguage) return;

  setCurrentLanguage(lang);
  applyStaticTranslations();
  markLanguageButtons();
  renderStatus();

  if (session.payload) renderRecentMenu(session.payload.recent || []);
  if (config.payload) {
    // The form is rebuilt from the payload, so anything typed and not yet saved
    // has to be folded back into it first. Changing language must not quietly
    // undo the user's edits.
    const dirty = config.dirty;
    if (dirty) config.payload.config = collectConfig();
    renderConfig(config.payload);
    if (dirty) markDirty(true);
  }
  if (devices.payload) renderDevices(devices.payload);
  if (diagrams.payload) renderDiagrams(diagrams.payload);
  if (exportTargets.payload) renderExportTargets(exportTargets.payload);
  if (lastSummary) renderSummary(lastSummary.summary, lastSummary.cancelled);
  renderAbout();

  // The export log is left as it is: it holds the pipeline's own output, which
  // is English in the GUI and on the command line alike.

  try {
    await postJSON("/api/language", { language: currentLanguage });
  } catch (err) {
    setStatus("msg.languageFailed", { error: err.message }, true);
  }
}

function renderRecentMenu(recent) {
  const menu = document.getElementById("config-menu");
  menu.replaceChildren();

  const current = session.payload?.configPath || "";
  const others = recent.filter((entry) => entry.path !== current);

  if (others.length) {
    menu.append(el("p", "config-menu-title", t("picker.recent")));
    for (const entry of others) {
      menu.append(recentItem(entry));
    }
    menu.append(el("div", "config-menu-separator"));
  }

  menu.append(menuAction(t("picker.open"), () => switchConfig("/api/config/open")));
  menu.append(menuAction(t("picker.new"), () => switchConfig("/api/config/new")));
  menu.append(menuAction(t("picker.reload"), () => switchConfig("/api/config/reload")));
}

function recentItem(entry) {
  const item = el("button", "config-menu-item");
  item.type = "button";
  item.title = entry.path;
  item.append(el("span", "config-menu-name", entry.name));
  item.append(el("span", "config-menu-dir", entry.dir));

  // A file that has been moved or deleted is still worth showing: it tells the
  // user where their profile used to be, but there is nothing to open.
  if (entry.missing) {
    item.classList.add("is-missing");
    item.disabled = true;
    item.append(el("span", "config-menu-note", t("picker.missing")));
    return item;
  }

  item.addEventListener("click", () => switchConfig("/api/config/open", entry.path));
  return item;
}

function menuAction(label, handler) {
  const item = el("button", "config-menu-item config-menu-action", label);
  item.type = "button";
  item.addEventListener("click", handler);
  return item;
}

export function toggleConfigMenu(open) {
  const menu = document.getElementById("config-menu");
  const shown = open === undefined ? menu.hidden : open;
  menu.hidden = !shown;
  document.getElementById("config-current").setAttribute("aria-expanded", String(shown));
}

// switchConfig changes the configuration the whole application works on.
export async function switchConfig(route, path) {
  toggleConfigMenu(false);

  if (config.dirty && !(await confirmDiscard())) return;

  try {
    const payload = await postJSON(route, path ? { path } : {});
    // The picker was dismissed: nothing changed, so say nothing.
    if (payload.cancelled) return;

    applySession(payload);
    resetAfterSwitch();
    setStatus("msg.configSwitched", { path: payload.configPath });
  } catch (err) {
    setStatus("msg.configSwitchFailed", { error: err.message }, true);
  }
}

// resetAfterSwitch drops everything the page holds about the previous
// configuration. Without it, the tabs already visited would keep showing the old
// profile's devices and diagrams until the application was restarted.
function resetAfterSwitch() {
  config.payload = null;
  markDirty(false);
  devices.payload = null;
  devices.expanded = null;
  diagrams.payload = null;

  // The log and the summary describe an export of the previous configuration,
  // and the preview points at a file resolved under the previous roots.
  document.getElementById("export-log").textContent = "";
  document.getElementById("export-summary").replaceChildren();
  document.getElementById("export-warnings").replaceChildren();
  closePreview();

  const active = document.querySelector('.tab[aria-selected="true"]')?.dataset.panel;
  opened.clear();
  loadConfig();
  if (active && active !== "configuration") {
    opened.add(active);
    onFirstOpen[active]?.();
  }
}

// confirmDiscard asks before throwing away unsaved edits, and resolves to false
// if the user would rather keep them.
function confirmDiscard() {
  return askConfirm(t("confirm.discard"), t("confirm.continue"));
}
