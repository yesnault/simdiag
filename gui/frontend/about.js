// The About tab: which version this is, whether a newer one exists, and
// installing it.
import { api, el, setStatus, postJSON, externalLink } from "./core.js";
import { t, currentLanguage } from "./i18n.js";

// The update block is the one part of this tab that is not static markup: it
// depends on what GitHub answered, so it is built here and re-rendered on a
// change of language, like the Devices and Diagrams panels.
const about = {
  payload: null,   // the last /api/update/check answer
  phase: "idle",   // idle | checking | installing | installed | failed | cancelled
  detail: null,    // {error} for a failure
};

export function renderAbout() {
  const node = document.getElementById("about-update");
  if (!node) return;
  node.replaceChildren();

  switch (about.phase) {
    case "checking":
      node.append(el("p", "about-note", t("about.checking")));
      return;
    case "installing":
      node.append(el("p", "about-note", t("about.installing")));
      node.append(aboutButton(t("generate.cancel"), cancelUpdate));
      return;
    case "installed":
      node.append(banner(t("about.installed", { version: about.detail.version })));
      node.append(aboutButton(t("about.restart"), restartSimDiag, "btn-primary"));
      return;
    case "failed":
      node.append(el("p", "about-note is-error", t("about.installFailed", { error: about.detail.error })));
      node.append(aboutButton(t("about.checkNow"), () => loadUpdateCheck(true)));
      return;
    case "cancelled":
      node.append(el("p", "about-note", t("about.installCancelled")));
      node.append(aboutButton(t("about.checkNow"), () => loadUpdateCheck(true)));
      return;
  }

  const payload = about.payload;
  if (!payload) {
    node.append(aboutButton(t("about.checkNow"), () => loadUpdateCheck(true)));
    return;
  }

  if (payload.development) {
    node.append(el("p", "about-note", t("about.developmentBuild", { version: payload.current })));
    return;
  }

  if (payload.checkFailed) {
    node.append(el("p", "about-note", t("about.checkFailed")));
    node.append(aboutButton(t("about.checkNow"), () => loadUpdateCheck(true)));
    return;
  }

  if (!payload.available) {
    const version = payload.latest ? payload.latest.version : payload.current;
    node.append(el("p", "about-note", t("about.upToDate", { version })));
    node.append(aboutButton(t("about.checkNow"), () => loadUpdateCheck(true)));
    return;
  }

  const latest = payload.latest;
  node.append(banner(t("about.available", { version: latest.version }),
    aboutButton(t("about.install"), installUpdate, "btn-primary")));

  if (latest.publishedAt) {
    node.append(el("p", "about-note", t("about.published", { date: formatDate(latest.publishedAt) })));
  }

  if (latest.notes) {
    node.append(el("h3", "about-subtitle", t("about.notesTitle")));
    // Preformatted, not rendered: the release body is Markdown written by hand,
    // and nothing in this frontend uses innerHTML.
    node.append(el("pre", "about-notes", latest.notes));
  }

  if (latest.url) {
    node.append(externalLink({ label: t("about.seeRelease"), target: "release" }));
  }
}

// banner is the amber "something is available" row, with its action at the end.
function banner(text, action) {
  const node = el("div", "warning-banner banner-action");
  node.append(el("span", null, text));
  if (action) node.append(action);
  return node;
}

function aboutButton(label, onClick, variant) {
  const button = el("button", `btn btn-small ${variant || ""}`.trim(), label);
  button.type = "button";
  button.addEventListener("click", onClick);
  return button;
}

// formatDate renders an RFC3339 stamp in the current language. Go sends the
// stamp rather than a formatted date, so the two languages can differ.
function formatDate(stamp) {
  const date = new Date(stamp);
  if (Number.isNaN(date.getTime())) return stamp;
  return date.toLocaleDateString(currentLanguage, { year: "numeric", month: "long", day: "numeric" });
}

// loadUpdateCheck asks the backend, which serves a remembered answer unless
// forced. A failure is reported inside the payload, not thrown: this also runs
// unasked at startup.
export async function loadUpdateCheck(force) {
  if (force) {
    about.phase = "checking";
    renderAbout();
  }

  try {
    about.payload = await api(`/api/update/check${force ? "?force=1" : ""}`);
  } catch (err) {
    about.payload = { checkFailed: err.message };
  }

  about.phase = "idle";
  markUpdateBadge(Boolean(about.payload && about.payload.available));
  renderAbout();
}

// markUpdateBadge puts a dot on the About tab, so a new version is noticed
// without the tab having to be opened.
function markUpdateBadge(available) {
  const tab = document.querySelector('.tab[data-panel="about"]');
  if (tab) tab.classList.toggle("has-badge", available);
}

async function installUpdate() {
  about.phase = "installing";
  about.detail = { version: about.payload.latest.version };
  renderAbout();

  const log = document.getElementById("update-log");
  log.textContent = "";
  log.hidden = false;

  try {
    await postJSON("/api/update/install", {});
  } catch (err) {
    about.phase = "failed";
    about.detail = { error: err.message };
    renderAbout();
    return;
  }

  pollUpdate();
}

// pollUpdate follows the run the same way pollExport does: Wails' asset server
// buffers a handler's response until it returns, so a streaming response would
// deliver nothing until the install was already over.
async function pollUpdate() {
  const log = document.getElementById("update-log");
  let from = 0;

  for (;;) {
    let state;
    try {
      state = await api(`/api/update/state?from=${from}`);
    } catch (err) {
      about.phase = "failed";
      about.detail = { error: err.message };
      renderAbout();
      return;
    }

    if (state.lines.length) {
      log.textContent += `${state.lines.join("\n")}\n`;
      log.scrollTop = log.scrollHeight;
    }
    from = state.nextIndex;

    if (!state.running) {
      if (state.cancelled) about.phase = "cancelled";
      else if (state.error) {
        about.phase = "failed";
        about.detail = { error: state.error };
      } else {
        about.phase = "installed";
        markUpdateBadge(false);
      }
      renderAbout();
      return;
    }

    await new Promise((resolve) => setTimeout(resolve, 250));
  }
}

async function cancelUpdate() {
  try {
    await postJSON("/api/update/cancel", {});
  } catch (err) {
    setStatus("msg.cancelFailed", { error: err.message }, true);
  }
}

async function restartSimDiag() {
  try {
    await postJSON("/api/update/restart", {});
  } catch (err) {
    setStatus("msg.restartFailed", { error: err.message }, true);
  }
}
