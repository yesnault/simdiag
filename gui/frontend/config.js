// The Configuration tab: the form projecting common.Config onto path fields,
// and reading it back.
import { api, el, setStatus, postJSON, externalLink } from "./core.js";
import { t, tMessage } from "./i18n.js";

export const config = {
  payload: null,
  dirty: false,
};

export function markDirty(dirty) {
  config.dirty = dirty;
  document.getElementById("dirty-marker").hidden = !dirty;
  document.getElementById("btn-save").disabled = !dirty;
}

// severityOf maps a pathStatus onto the dot colour next to a field.
function severityOf(status) {
  if (!status) return "";
  if (status.severity) return status.severity;
  return status.exists ? "ok" : "";
}

// pathField builds one labelled path input with a Browse button, a status dot
// and, when the field is empty, a one-click suggestion.
//
// spec.help is the sentence saying what the field is for. It is permanent and
// neutral, unlike the status line below it: a pilot who has never read the
// README should not have to guess what a template is, or which of the many DCS
// folders is the one being asked for.
function pathField(spec) {
  const { label, name, value, status, kind, filter, hint, help, link, placeholder } = spec;

  const field = el("div", "field field-path");
  field.dataset.severity = severityOf(status);

  const labelRow = el("div", "field-label-row");
  labelRow.append(el("label", "field-label", label));
  if (hint) labelRow.append(el("span", "field-hint", hint));
  field.append(labelRow);

  const row = el("div", "field-row");

  const input = el("input", "field-input");
  input.type = "text";
  input.name = name;
  input.value = value || "";
  if (placeholder) input.placeholder = placeholder;
  input.addEventListener("input", () => markDirty(true));
  row.append(input);

  const browse = el("button", "btn btn-browse", t("config.browse"));
  browse.type = "button";
  browse.addEventListener("click", async () => {
    try {
      const res = await api("/api/browse", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          kind: kind || "folder",
          title: label,
          current: input.value,
          filter: filter || "",
        }),
      });
      // An empty path means the picker was cancelled.
      if (res.path) {
        input.value = res.path;
        markDirty(true);
      }
    } catch (err) {
      setStatus("msg.pickerFailed", { error: err.message }, true);
    }
  });
  row.append(browse);

  field.append(row);

  if (help) field.append(el("p", "field-help", help));
  if (link) field.append(externalLink(link));

  if (spec.suggestion && !value) {
    const suggest = el("button", "field-suggestion", t("config.useSuggestion", { path: spec.suggestion }));
    suggest.type = "button";
    suggest.addEventListener("click", () => {
      input.value = spec.suggestion;
      markDirty(true);
    });
    field.append(suggest);
  }

  if (status && status.detail && status.detail.code) {
    field.append(el("p", "field-detail", tMessage(status.detail)));
  }

  return field;
}

function section(title, description) {
  const node = el("section", "config-section");
  const header = el("header", "config-section-header");
  header.append(el("h2", null, title));
  if (description) header.append(el("p", null, description));
  node.append(header);
  return node;
}

// subsection groups one simulator inside a section covering several, which is
// what the two IL-2 titles need: they are configured separately but share the
// SimpleRadio installation shown alongside them.
function subsection(title) {
  const node = el("div", "config-subsection");
  node.append(el("h3", "config-subsection-title", title));
  return node;
}

// srsField builds the SimpleRadio path input. There are two installations,
// DCS-SRS and IL2-SRS, not one per simulator.
function srsField(name, value, status, suggestion, hint) {
  return pathField({
    label: t("config.srs"),
    name,
    value,
    status,
    suggestion,
    hint,
    help: t("config.srsHelp"),
    placeholder: t("config.srsPlaceholder"),
  });
}

// SIMULATOR_HELP says where each simulator keeps its files, which is exactly
// what nobody guesses: DCS is under Saved Games, IL-2 under its install folder.
//
// The keys are spelled out one per simulator rather than composed from the key,
// because gui/i18n_test.go only recognises a literal key inside the call. A computed key
// would make every one of these entries look unused and fail the build.
const SIMULATOR_HELP = {
  dcs_world: () => t("config.dcsPathHelp"),
  il2_sturmovik: () => t("config.il2PathHelp"),
  il2_korea: () => t("config.il2KoreaHelp"),
};

// baseTemplatesBanner offers the templates SimDiag ships with. Without any, the
// tool has nothing to draw on and every export is refused, so this comes before
// the fields, and it is an offer rather than something done behind the user's
// back: templates are meant to be edited, and what is on disk always wins.
function baseTemplatesBanner(base) {
  const banner = el("div", "warning-banner banner-action");
  banner.append(el("span", null,
    t("templates.bannerMissing", { n: base.missing.length, path: base.target })));

  const install = el("button", "btn btn-small", t("templates.bannerInstall"));
  install.type = "button";
  install.addEventListener("click", async () => {
    install.disabled = true;
    try {
      const result = await postJSON("/api/templates/install", {});
      renderConfig(result.config);
      setStatus("msg.templatesInstalled", { n: result.installed, path: result.target });
    } catch (err) {
      install.disabled = false;
      setStatus("msg.templatesInstallFailed", { error: err.message }, true);
    }
  });
  banner.append(install);

  return banner;
}

export function renderConfig(payload) {
  config.payload = payload;

  const form = document.getElementById("config-form");
  form.replaceChildren();

  const cfg = payload.config;
  const defaults = payload.defaults;
  const status = payload.status;

  // --- what the binary can add before anything else is worth configuring
  const base = status.baseTemplates;
  if (base && base.missing.length) {
    form.append(baseTemplatesBanner(base));
  }

  // --- general
  const general = section(t("config.general"), t("config.generalNote"));
  general.append(
    pathField({
      label: t("config.templatesDirectory"),
      name: "templatesDirectory",
      value: cfg.templatesDirectory,
      status: status.templates,
      suggestion: defaults.templatesDirectory,
      help: t("config.templatesHelp"),
      placeholder: t("config.templatesPlaceholder"),
    }),
    pathField({
      label: t("config.outputDirectory"),
      name: "outputDirectory",
      value: cfg.outputDirectory,
      status: status.output,
      suggestion: defaults.outputDirectory,
      help: t("config.outputHelp"),
      placeholder: t("config.outputPlaceholder"),
    }),
  );
  form.append(general);

  // --- optional tools
  const tools = section(t("config.tools"), t("config.toolsNote"));
  tools.append(
    pathField({
      label: "draw.io",
      name: "drawioPath",
      value: cfg.drawioPath,
      status: status.drawio,
      kind: "file",
      filter: "*.exe",
      hint: t("config.drawioHint"),
      help: t("config.drawioHelp"),
      link: { label: "drawio.com", target: "drawio" },
      suggestion: defaults.drawioPath,
      placeholder: "draw.io.exe",
    }),
    pathField({
      label: t("config.openkneeboard"),
      name: "openkneeboardProfilesFilepath",
      value: cfg.openkneeboardProfilesFilepath,
      status: status.openkneeboard,
      kind: "file",
      filter: "*.json",
      help: t("config.openkneeboardHelp"),
      link: { label: "openkneeboard.com", target: "openkneeboard" },
      suggestion: defaults.openkneeboardProfilesFilepath,
      placeholder: "Profiles.json",
    }),
  );
  form.append(tools);

  // --- DCS, then the two IL-2 titles under one section
  const simulatorOf = (key) => cfg.simulators.find((s) => s.key === key);

  const dcs = simulatorOf("dcs_world");
  if (dcs) {
    const node = section(dcs.label);
    node.dataset.simulator = dcs.key;

    const dcsStatus = status.simulators[dcs.key] || {};
    if (dcsStatus.modules && dcsStatus.modules.length) {
      node.querySelector(".config-section-header").append(
        el("p", "config-section-note",
          t("config.detectedModules", { modules: dcsStatus.modules.join(", ") })),
      );
    }

    node.append(...simulatorFields(dcs, dcsStatus, defaultsFor(defaults, dcs.key)));
    node.append(srsField("dcsSrsPath", cfg.dcsSrsPath, status.dcsSrs, defaults.dcsSrsPath));
    form.append(node);
  }

  // One SimpleRadio installation serves both IL-2 titles, so the two are
  // configured inside a single section with the shared SRS path underneath.
  const il2Keys = ["il2_sturmovik", "il2_korea"].filter(simulatorOf);
  if (il2Keys.length) {
    const node = section("IL-2");

    for (const key of il2Keys) {
      const sim = simulatorOf(key);
      const block = subsection(sim.label);
      block.dataset.simulator = key;
      block.append(...simulatorFields(sim, status.simulators[key] || {}, defaultsFor(defaults, key)));
      node.append(block);
    }

    node.append(
      srsField("il2SrsPath", cfg.il2SrsPath, status.il2Srs, defaults.il2SrsPath,
        t("config.srsShared")),
    );
    form.append(node);
  }

  markDirty(false);
}

function defaultsFor(defaults, key) {
  return defaults.simulators.find((d) => d.key === key) || {};
}

// simulatorFields builds what belongs to one simulator alone. SimpleRadio is not
// part of it: there are two installations, not one per simulator.
function simulatorFields(sim, simStatus, simDefaults) {
  return [
    pathField({
      label: t("config.simulatorFiles"),
      name: `${sim.key}.path`,
      value: sim.path,
      status: simStatus.path,
      suggestion: simDefaults.path,
      help: SIMULATOR_HELP[sim.key]?.(),
      placeholder: t("config.simulatorPlaceholder"),
    }),
    pathField({
      label: t("config.gremlins"),
      name: `${sim.key}.gremlinsProfileFilepath`,
      value: sim.gremlinsProfileFilepath,
      status: simStatus.gremlins,
      kind: "file",
      filter: "*.xml",
      help: t("config.gremlinsHelp"),
    }),
    pathField({
      label: t("config.target"),
      name: `${sim.key}.targetProfileFilepath`,
      value: sim.targetProfileFilepath,
      status: simStatus.target,
      kind: "file",
      filter: "*.tmc",
      help: t("config.targetHelp"),
    }),
  ];
}

// collectConfig reads the form back into the shape the API expects.
export function collectConfig() {
  const form = document.getElementById("config-form");
  const value = (name) => {
    const node = form.elements[name];
    return node ? node.value.trim() : "";
  };

  return {
    templatesDirectory: value("templatesDirectory"),
    outputDirectory: value("outputDirectory"),
    drawioPath: value("drawioPath"),
    openkneeboardProfilesFilepath: value("openkneeboardProfilesFilepath"),
    dcsSrsPath: value("dcsSrsPath"),
    il2SrsPath: value("il2SrsPath"),
    simulators: config.payload.config.simulators.map((sim) => ({
      key: sim.key,
      label: sim.label,
      path: value(`${sim.key}.path`),
      gremlinsProfileFilepath: value(`${sim.key}.gremlinsProfileFilepath`),
      targetProfileFilepath: value(`${sim.key}.targetProfileFilepath`),
    })),
  };
}

export async function loadConfig() {
  try {
    renderConfig(await api("/api/config"));
    setStatus("msg.ready");
  } catch (err) {
    setStatus("msg.configLoadFailed", { error: err.message }, true);
  }
}

export async function saveConfig() {
  const button = document.getElementById("btn-save");
  button.disabled = true;
  try {
    renderConfig(
      await api("/api/config", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(collectConfig()),
      }),
    );
    setStatus("msg.configSaved");
  } catch (err) {
    setStatus("msg.saveFailed", { error: err.message }, true);
    button.disabled = false;
  }
}
