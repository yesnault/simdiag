// What every tab needs: the HTTP helpers, the DOM helper, the footer status,
// the tab strip, the confirmation modal and the external links.
//
// Nothing here knows about any particular tab. Everything else imports from it.
import { t, tMessage } from "./i18n.js";

// The frontend talks to Go over plain HTTP served by the asset handler, so
// there is no build step and no generated bindings to keep in sync.
export async function api(path, options) {
  const res = await fetch(path, options);
  if (!res.ok) {
    throw new Error(await errorText(res));
  }
  return res.json();
}

// errorText reads a failed response. Refusals the user is meant to act on come
// back as a {code, args} message so they can be shown in their language; the
// rest is technical text from Go, kept verbatim.
async function errorText(res) {
  const body = (await res.text()).trim();

  try {
    const parsed = JSON.parse(body);
    if (parsed && parsed.code) return tMessage(parsed);
  } catch {
    // Not JSON: plain text, or nothing at all.
  }

  return body || `${res.status} ${res.statusText}`;
}

export async function postJSON(path, body) {
  return api(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

// lastStatus remembers what the footer says, so the message survives a change of
// language instead of being left behind in the previous one.
let lastStatus = null;

export function setStatus(key, params, isError) {
  lastStatus = key ? { key, params, isError } : null;
  renderStatus();
}

export function renderStatus() {
  const node = document.getElementById("status");
  node.textContent = lastStatus ? t(lastStatus.key, lastStatus.params) : "";
  node.classList.toggle("is-error", Boolean(lastStatus && lastStatus.isError));
}

export function el(tag, className, text) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

// onFirstOpen holds work to run the first time a tab is shown, so a tab that
// needs to hit the disk does not pay for it until the user asks for it.
export const onFirstOpen = {};
export const opened = new Set();

export function activateTab(name) {
  for (const tab of document.querySelectorAll(".tab")) {
    const selected = tab.dataset.panel === name;
    tab.setAttribute("aria-selected", String(selected));
    document
      .getElementById(`panel-${tab.dataset.panel}`)
      .classList.toggle("is-active", selected);
  }

  if (!opened.has(name)) {
    opened.add(name);
    onFirstOpen[name]?.();
  }
}

export function setupTabs() {
  for (const tab of document.querySelectorAll(".tab")) {
    tab.addEventListener("click", () => activateTab(tab.dataset.panel));
  }

  // A hash selects the starting tab, which also makes the window deep-linkable.
  const initial = location.hash.replace("#", "");
  activateTab(document.getElementById(`panel-${initial}`) ? initial : "configuration");
}

// -------------------------------------------------------------- external links

// A plain <a href> would navigate this window away from the interface, with no
// way back: the webview is a browser and the application is the page it shows.
// So a link is a button, and Go hands the URL to the system browser. The page
// names a destination rather than a URL. The table on the Go side is the whole
// set of pages the interface will open (gui/tips_routes.go).
export async function openExternal(target) {
  try {
    await postJSON("/api/open-url", { target });
  } catch (err) {
    setStatus("msg.linkFailed", { error: err.message }, true);
  }
}

// externalLink builds the link-looking button the configuration form uses. The
// Tips tab spells its own out in the markup, with the same data-link attribute.
export function externalLink(spec) {
  const button = el("button", "field-link", spec.label);
  button.type = "button";
  button.dataset.link = spec.target;
  return button;
}


// askConfirm shows the modal and resolves to what the user answered. It takes
// rendered sentences rather than catalogue keys: gui/i18n_test.go only sees
// the key spelled out inside the call, so the keys stay at the call sites.
export function askConfirm(message, okLabel) {
  const backdrop = document.getElementById("confirm");
  document.getElementById("confirm-message").textContent = message;
  document.getElementById("confirm-ok").textContent = okLabel;
  backdrop.hidden = false;

  return new Promise((resolve) => {
    const finish = (answer) => {
      backdrop.hidden = true;
      document.getElementById("confirm-ok").removeEventListener("click", onOk);
      document.getElementById("confirm-cancel").removeEventListener("click", onCancel);
      resolve(answer);
    };
    const onOk = () => finish(true);
    const onCancel = () => finish(false);

    document.getElementById("confirm-ok").addEventListener("click", onOk);
    document.getElementById("confirm-cancel").addEventListener("click", onCancel);
  });
}
