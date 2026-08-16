// The Diagrams tab and the image preview it opens.
//
// They share a file because they call each other: a diagram card opens the
// preview, and the preview asks the tab how to label the path it is showing.
import { api, el, setStatus, postJSON, activateTab } from "./core.js";
import { setExporting, pollExport } from "./generate.js";
import { t, tMessage } from "./i18n.js";

export const diagrams = { payload: null };

export async function loadDiagrams() {
  try {
    renderDiagrams(await api("/api/diagrams"));
  } catch (err) {
    setStatus("msg.diagramsReadFailed", { error: err.message }, true);
  }
}

// displayPathFor picks what to show for a diagram.
//
// The generated SVG carries its labels as draw.io-escaped HTML, which only
// draw.io itself re-parses: a browser shows "&lt;div&gt;Volets - Rétraction&lt;/div&gt;"
// verbatim. The PNG that draw.io exported alongside renders those labels
// properly, so it is what we display whenever it exists.
function displayPathFor(diagram) {
  return diagram.pngPath || diagram.path;
}

function diagramCard(diagram) {
  const card = el("button", "diagram-card");
  card.type = "button";

  const thumb = el("div", "diagram-thumb");
  const img = el("img");
  // Thumbnails come straight from disk by URL; a 1.6 MB file never goes through
  // the JSON bridge.
  img.src = `/api/diagram?p=${encodeURIComponent(displayPathFor(diagram))}`;
  img.loading = "lazy";
  img.alt = "";
  thumb.append(img);
  card.append(thumb);

  const meta = el("div", "diagram-meta");
  meta.append(el("span", "diagram-name", diagram.name));

  const detail = el("span", "diagram-detail");
  detail.textContent = `${Math.round(diagram.size / 1024)} KB · ${diagram.modTime}`;
  if (!diagram.pngPath) detail.textContent += t("diagrams.noPNG");
  meta.append(detail);
  card.append(meta);

  card.addEventListener("click", () => openDiagram(diagram));
  return card;
}

export function renderDiagrams(payload) {
  const body = document.getElementById("diagrams-body");
  body.replaceChildren();

  for (const warning of payload.warnings || []) {
    body.append(el("p", "warning-banner", tMessage(warning)));
  }

  let total = 0;
  for (const group of payload.groups) {
    total += group.diagrams.length;

    const node = el("section", "diagram-group");
    const header = el("header", "diagram-group-header");
    // A simulator or module name is a proper noun; only the output directory
    // itself is a phrase, and it arrives as a code.
    header.append(el("h2", null, group.labelCode ? t(group.labelCode) : group.label));
    header.append(el("span", "diagram-count", t("diagrams.groupCount", { n: group.diagrams.length })));

    const open = el("button", "btn btn-small", t("diagrams.openFolder"));
    open.type = "button";
    open.addEventListener("click", () => openFolder(group.diagrams[0].path));
    header.append(open);
    node.append(header);

    const grid = el("div", "diagram-grid");
    for (const diagram of group.diagrams) grid.append(diagramCard(diagram));
    node.append(grid);

    body.append(node);
  }

  document.getElementById("diagrams-summary").textContent = total
    ? t("diagrams.summary", { total, path: payload.outputPath })
    : "";
  document.getElementById("btn-regenerate").disabled = !payload.hasCsv;
  document.getElementById("btn-open-output").disabled = !payload.outputPath;

  diagrams.payload = payload;
}

export async function openFolder(path) {
  try {
    await postJSON("/api/diagrams/open", { path });
  } catch (err) {
    setStatus("msg.folderOpenFailed", { error: err.message }, true);
  }
}

export async function regenerateDiagrams() {
  const button = document.getElementById("btn-regenerate");
  button.disabled = true;
  // Progress shows on the Generate tab, which already has the log and summary.
  activateTab("generate");
  document.getElementById("export-log").textContent = "";
  document.getElementById("export-summary").replaceChildren();
  setExporting(true);
  setStatus("msg.regenerating");

  try {
    const initial = await api("/api/diagrams/regenerate", { method: "POST" });
    await pollExport(initial.nextIndex || 0);
    await loadDiagrams();
  } catch (err) {
    setStatus("msg.regenerationFailed", { error: err.message }, true);
    setExporting(false);
  } finally {
    button.disabled = false;
  }
}

// -------------------------------------------------------------- preview

// Templates and diagrams are 280 KB to 1.6 MB, so the preview loads them by URL
// rather than pushing the markup through JSON.
const preview = { scale: 1, x: 0, y: 0, dragging: false, startX: 0, startY: 0, path: null };

function applyPreviewTransform() {
  const img = document.getElementById("preview-image");
  img.style.transform = `translate(${preview.x}px, ${preview.y}px) scale(${preview.scale})`;
  document.getElementById("preview-zoom").textContent = `${Math.round(preview.scale * 100)}%`;
}

function resetPreview() {
  preview.scale = 1;
  preview.x = 0;
  preview.y = 0;
  applyPreviewTransform();
}

function zoomPreview(factor) {
  preview.scale = Math.min(8, Math.max(0.1, preview.scale * factor));
  applyPreviewTransform();
}

function showPreview(url, title, path) {
  document.getElementById("preview-title").textContent = title;
  document.getElementById("preview-image").src = url;
  document.getElementById("preview").hidden = false;
  preview.path = path || null;
  resetPreview();
}

export function openPreview(path, name) {
  showPreview(`/api/template?p=${encodeURIComponent(path)}`, name || path, null);
}

export function openDiagram(diagram) {
  const path = displayPathFor(diagram);
  const title = diagram.pngPath
    ? diagram.name
    : t("diagrams.svgNote", { name: diagram.name });
  showPreview(`/api/diagram?p=${encodeURIComponent(path)}`, title, diagram.path);
}

export function closePreview() {
  document.getElementById("preview").hidden = true;
  document.getElementById("preview-image").src = "";
  preview.path = null;
}

export function setupPreviewControls() {
  const stage = document.getElementById("preview-stage");
  const img = document.getElementById("preview-image");

  document.getElementById("preview-in").addEventListener("click", () => zoomPreview(1.25));
  document.getElementById("preview-out").addEventListener("click", () => zoomPreview(0.8));
  document.getElementById("preview-fit").addEventListener("click", resetPreview);

  stage.addEventListener("wheel", (e) => {
    e.preventDefault();
    zoomPreview(e.deltaY < 0 ? 1.12 : 0.89);
  }, { passive: false });

  img.addEventListener("pointerdown", (e) => {
    preview.dragging = true;
    preview.startX = e.clientX - preview.x;
    preview.startY = e.clientY - preview.y;
    img.setPointerCapture(e.pointerId);
  });
  img.addEventListener("pointermove", (e) => {
    if (!preview.dragging) return;
    preview.x = e.clientX - preview.startX;
    preview.y = e.clientY - preview.startY;
    applyPreviewTransform();
  });
  img.addEventListener("pointerup", (e) => {
    preview.dragging = false;
    img.releasePointerCapture(e.pointerId);
  });
}
