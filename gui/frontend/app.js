// Entry point: the only script index.html loads. It wires the markup to the
// modules and starts the application.
import { onFirstOpen, setupTabs, openExternal, askConfirm } from "./core.js";
import { loadConfig, saveConfig } from "./config.js";
import { loadSession, changeLanguage, toggleConfigMenu, switchConfig } from "./session.js";
import { scanDevices, detectTargetNumbers } from "./devices.js";
import { loadDiagrams, regenerateDiagrams, openFolder, closePreview, setupPreviewControls } from "./diagrams.js";
import { loadExportTargets, runExport, cancelExport } from "./generate.js";
import { createBatchScript } from "./tips.js";
import { renderAbout, loadUpdateCheck } from "./about.js";

// Opening the Devices tab re-parses the simulator configurations; doing it on
// first open beats showing an empty tab with a button on it.
onFirstOpen.devices = scanDevices;
onFirstOpen.generate = loadExportTargets;
onFirstOpen.diagrams = loadDiagrams;
onFirstOpen.about = renderAbout;

document.getElementById("btn-save").addEventListener("click", saveConfig);
document.getElementById("btn-reload").addEventListener("click", loadConfig);
document.getElementById("btn-scan").addEventListener("click", scanDevices);
document.getElementById("btn-detect-target").addEventListener("click", detectTargetNumbers);
document.getElementById("btn-export").addEventListener("click", runExport);
document.getElementById("btn-cancel").addEventListener("click", cancelExport);
document.getElementById("preview-close").addEventListener("click", closePreview);
document.getElementById("btn-refresh-diagrams").addEventListener("click", loadDiagrams);
document.getElementById("btn-regenerate").addEventListener("click", regenerateDiagrams);
// The toolbar button opens the output directory named beside it, not the first
// module's folder. A group's own button is what opens that.
document.getElementById("btn-open-output").addEventListener("click", () => openFolder("."));
document.getElementById("config-current").addEventListener("click", (e) => {
  e.stopPropagation();
  toggleConfigMenu();
});
document.getElementById("btn-config-open").addEventListener("click", () =>
  switchConfig("/api/config/open"),
);
// Any click outside the menu closes it, which is what a dropdown is expected to
// do and saves the menu from needing its own dismiss button.
document.addEventListener("click", (e) => {
  if (!e.target.closest("#config-menu")) toggleConfigMenu(false);
});
document.addEventListener("keydown", (e) => {
  if (e.key !== "Escape") return;
  closePreview();
  toggleConfigMenu(false);
});

document.getElementById("btn-create-bat").addEventListener("click", createBatchScript);
// Delegated, because the configuration form builds its links after this runs.
document.addEventListener("click", (e) => {
  const link = e.target.closest("[data-link]");
  if (link) openExternal(link.dataset.link);
});

document.getElementById("config-menu").addEventListener("click", (e) => e.stopPropagation());
for (const button of document.querySelectorAll(".lang")) {
  button.addEventListener("click", () => changeLanguage(button.dataset.language));
}

setupPreviewControls();

// The language arrives with the session, so it has to be known before the first
// panel is drawn. Otherwise the form renders in English and flips a moment
// later.
(async () => {
  await loadSession();
  setupTabs();
  loadConfig();

  // Deliberately not awaited: the badge appears when GitHub answers, and a slow
  // or missing network must not hold up the interface. The backend serves a
  // remembered answer for six hours, so this is usually free.
  loadUpdateCheck(false);
})();
