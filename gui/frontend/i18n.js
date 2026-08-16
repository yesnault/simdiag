// The interface is bilingual and every sentence lives here, both languages side
// by side. Go never formats prose: it sends a message code plus its values (see
// gui/message.go), so switching language only re-renders what is already loaded.
//
// French follows the vocabulary flight simmers actually use: a device is a
// "contrôleur", but "binding" and "template" stay as they are. Product names are
// never translated: DCS World, IL-2, draw.io, OpenKneeboard, SimpleRadio,
// TARGET, Joystick Gremlins.
//
// Plurals are written "(s)" in both languages, so no plural rules are needed;
// the French entries carry their own agreement.
const MESSAGES = {
  // ---------------------------------------------------------------- chrome
  "app.configLabel": { en: "config", fr: "config" },
  "app.configTitle": {
    en: "Configuration file in use — click for recent files",
    fr: "Fichier de configuration utilisé — cliquer pour les fichiers récents",
  },
  "app.open": { en: "Open…", fr: "Ouvrir…" },
  "app.language": { en: "Interface language", fr: "Langue de l'interface" },

  "tab.configuration": { en: "Configuration", fr: "Configuration" },
  "tab.devices": { en: "Devices", fr: "Contrôleurs" },
  "tab.generate": { en: "Generate", fr: "Générer" },
  "tab.diagrams": { en: "Diagrams", fr: "Diagrammes" },
  "tab.tips": { en: "Tips", fr: "Astuces" },
  "tab.about": { en: "About", fr: "À propos" },

  // ------------------------------------------------------ configuration tab
  "config.unsaved": { en: "Unsaved changes", fr: "Modifications non enregistrées" },
  "config.reload": { en: "Reload", fr: "Recharger" },
  "config.save": { en: "Save", fr: "Enregistrer" },
  "config.browse": { en: "Browse…", fr: "Parcourir…" },
  "config.useSuggestion": { en: "Use {path}", fr: "Utiliser {path}" },

  "templates.bannerMissing": {
    en: "{n} base template(s) shipped with SimDiag are not in {path}.",
    fr: "{n} template(s) de base fourni(s) avec SimDiag ne sont pas dans {path}.",
  },
  "templates.bannerInstall": {
    en: "Install the base templates",
    fr: "Installer les templates de base",
  },

  "config.general": { en: "General", fr: "Général" },
  "config.generalNote": { en: "Shared by every simulator.", fr: "Commun à tous les simulateurs." },
  "config.templatesDirectory": { en: "Templates directory", fr: "Répertoire des templates" },
  "config.templatesPlaceholder": {
    en: "Directory holding the SVG templates",
    fr: "Répertoire contenant les templates SVG",
  },
  "config.templatesHelp": {
    en: "A template is an SVG file drawing one controller. The ones SimDiag ships with can be edited and you can draw your own; a controller is paired with its template in the Devices tab.",
    fr: "Un template est un fichier SVG qui représente un contrôleur. Ceux fournis avec SimDiag sont modifiables et vous pouvez créer les vôtres ; l'association entre un contrôleur et son template se fait dans l'onglet Contrôleurs.",
  },
  "config.outputDirectory": { en: "Output directory", fr: "Répertoire de sortie" },
  "config.outputPlaceholder": { en: "Where diagrams are written", fr: "Où les diagrammes sont écrits" },
  "config.outputHelp": {
    en: "Directory SimDiag writes the generated SVG and PNG files to.",
    fr: "Répertoire dans lequel SimDiag écrit les fichiers SVG et PNG générés.",
  },

  "config.tools": { en: "Tools", fr: "Outils" },
  "config.toolsNote": { en: "Optional.", fr: "Optionnels." },
  "config.drawioHint": { en: "PNG export", fr: "export PNG" },
  "config.drawioHelp": {
    en: "draw.io converts the generated SVG files to PNG. It is also the tool to use for drawing and editing templates by hand, which is what you need for a controller no template covers yet.",
    fr: "draw.io convertit en PNG les fichiers SVG générés. C'est aussi le logiciel à utiliser pour créer et modifier les templates à la main, ce qu'il faut faire pour un contrôleur qu'aucun template ne couvre encore.",
  },
  "config.openkneeboard": { en: "OpenKneeboard profiles", fr: "Profils OpenKneeboard" },
  "config.openkneeboardHelp": {
    en: "OpenKneeboard shows the diagrams while you fly. Very handy in VR, it works without a headset too. SimDiag also reads its bindings from this file and adds them to the diagrams.",
    fr: "OpenKneeboard affiche les diagrammes pendant le vol. Très pratique en VR, il fonctionne aussi sans casque. SimDiag lit en plus les bindings déclarés dans ce fichier et les ajoute aux diagrammes.",
  },

  "config.simulatorFiles": { en: "Configuration files", fr: "Fichiers de configuration" },
  "config.simulatorPlaceholder": {
    en: "Leave empty to skip this simulator",
    fr: "Laisser vide pour ignorer ce simulateur",
  },
  "config.dcsPathHelp": {
    en: "DCS World keeps its controller configuration in Saved Games\\DCS (or Saved Games\\DCS.openbeta). Point at that directory: SimDiag detects the installed modules on its own, there is nothing to declare.",
    fr: "DCS World range la configuration des contrôleurs dans Saved Games\\DCS (ou Saved Games\\DCS.openbeta). Indiquez ce répertoire : SimDiag y détecte seul les modules installés, il n'y a rien à déclarer.",
  },
  "config.il2PathHelp": {
    en: "Point at the data\\input directory of the installation, for example C:\\Program Files\\IL-2 Sturmovik Great Battles\\data\\input.",
    fr: "Indiquez le répertoire data\\input de l'installation, par exemple C:\\Program Files\\IL-2 Sturmovik Great Battles\\data\\input.",
  },
  "config.il2KoreaHelp": {
    en: "Point at the data\\Input directory of the installation, for example C:\\Program Files\\IL2Series\\game\\data\\Input. Action names are borrowed from Great Battles when it is configured above.",
    fr: "Indiquez le répertoire data\\Input de l'installation, par exemple C:\\Program Files\\IL2Series\\game\\data\\Input. Les noms d'actions sont empruntés à Great Battles lorsqu'il est configuré ci-dessus.",
  },
  "config.gremlins": { en: "Joystick Gremlins profile", fr: "Profil Joystick Gremlins" },
  "config.gremlinsHelp": {
    en: "Joystick Gremlins remaps controllers onto virtual devices. SimDiag follows that remapping back to the physical button, so the binding lands on the right controller.",
    fr: "Joystick Gremlins remappe des contrôleurs vers des périphériques virtuels. SimDiag remonte ce remappage jusqu'au bouton physique, pour que le binding se pose sur le bon contrôleur.",
  },
  "config.target": { en: "TARGET profile", fr: "Profil TARGET" },
  "config.targetHelp": {
    en: "Thrustmaster TARGET profile. SimDiag also reads the keyboard layout declared in the file, which is how AZERTY keys are matched to the simulator's bindings.",
    fr: "Profil Thrustmaster TARGET. SimDiag y lit aussi la disposition de clavier déclarée dans le fichier, ce qui permet de faire correspondre les touches AZERTY aux bindings du simulateur.",
  },
  "config.srs": { en: "SimpleRadio (SRS)", fr: "SimpleRadio (SRS)" },
  "config.srsHelp": {
    en: "Adds the radio controls defined in SimpleRadio Standalone — push-to-talk, radio selection — to the diagrams.",
    fr: "Ajoute aux diagrammes les commandes radio définies dans SimpleRadio Standalone : alternat (PTT), sélection de radio.",
  },
  "config.srsPlaceholder": {
    en: "SimpleRadio-Standalone directory",
    fr: "Répertoire SimpleRadio-Standalone",
  },
  "config.srsShared": { en: "shared by both IL-2 titles", fr: "commun aux deux IL-2" },
  "config.detectedModules": { en: "Detected modules: {modules}", fr: "Modules détectés : {modules}" },

  // ------------------------------------------------- configuration statuses
  "status.notConfigured": { en: "Not configured", fr: "Non configuré" },
  "status.templates.required": {
    en: "Required: no diagram can be generated without templates",
    fr: "Obligatoire : aucun diagramme ne peut être généré sans templates",
  },
  "status.templates.dirNotFound": { en: "Directory not found", fr: "Répertoire introuvable" },
  "status.templates.unreadable": { en: "{error}", fr: "{error}" },
  "status.templates.none": {
    en: "No .svg template found in this directory",
    fr: "Aucun template .svg dans ce répertoire",
  },
  "status.templates.found": { en: "{count} template(s) found", fr: "{count} template(s) trouvé(s)" },
  "status.output.required": {
    en: "Required: nowhere to write the diagrams",
    fr: "Obligatoire : nulle part où écrire les diagrammes",
  },
  "status.output.willBeCreated": { en: "Will be created on export", fr: "Sera créé lors de l'export" },
  "status.drawio.notFound": {
    en: "draw.io not found: diagrams will be SVG only, no PNG export",
    fr: "draw.io introuvable : diagrammes en SVG uniquement, pas d'export PNG",
  },
  "status.openkneeboard.notFound": {
    en: "File not found: PTT bindings will not be added",
    fr: "Fichier introuvable : les bindings PTT ne seront pas ajoutés",
  },
  "status.srs.notFound": {
    en: "Directory not found: radio bindings will not be added",
    fr: "Répertoire introuvable : les bindings radio ne seront pas ajoutés",
  },
  "status.gremlins.notFound": {
    en: "File not found: Joystick Gremlins bindings will not be added",
    fr: "Fichier introuvable : les bindings Joystick Gremlins ne seront pas ajoutés",
  },
  "status.target.notFound": {
    en: "File not found: TARGET bindings will not be added",
    fr: "Fichier introuvable : les bindings TARGET ne seront pas ajoutés",
  },
  "status.simulator.notConfigured": {
    en: "Not configured: this simulator is skipped on export",
    fr: "Non configuré : ce simulateur sera ignoré à l'export",
  },
  "status.simulator.dirNotFound": { en: "Directory not found", fr: "Répertoire introuvable" },

  // ------------------------------------------------------------ devices tab
  "devices.rescan": { en: "Rescan", fr: "Analyser à nouveau" },
  "devices.detectTarget": { en: "Detect TARGET numbers", fr: "Détecter les numéros TARGET" },
  "devices.detectTargetTitle": {
    en: "Match the TARGET device numbers to your controllers by name",
    fr: "Faire correspondre par le nom les numéros de contrôleur TARGET aux vôtres",
  },
  "devices.loading": {
    en: "Reading the simulator configurations…",
    fr: "Lecture des configurations des simulateurs…",
  },
  "devices.summary": {
    en: "{n} device(s), {assigned} with a template, {total} template(s) available",
    fr: "{n} contrôleur(s), {assigned} avec un template, {total} template(s) disponible(s)",
  },
  "devices.none": {
    en: "No device found. Check the simulator paths in the Configuration tab.",
    fr: "Aucun contrôleur trouvé. Vérifiez les chemins des simulateurs dans l'onglet Configuration.",
  },
  "devices.virtual": { en: "virtual", fr: "virtuel" },
  "devices.bindings": { en: "{n} bindings", fr: "{n} bindings" },
  "devices.ignored": { en: "Ignored", fr: "Ignoré" },
  "devices.noTemplate": { en: "No template assigned", fr: "Aucun template attribué" },
  "devices.preview": { en: "Preview", fr: "Aperçu" },
  "devices.change": { en: "Change", fr: "Changer" },
  "devices.assign": { en: "Assign", fr: "Attribuer" },
  "devices.ignore": { en: "Ignore", fr: "Ignorer" },
  "devices.set": { en: "Set", fr: "Définir" },
  "devices.targetNumber": {
    en: "Thrustmaster TARGET device number",
    fr: "Numéro de contrôleur Thrustmaster TARGET",
  },
  "devices.noTemplateFound": {
    en: "No template found in the templates directory.",
    fr: "Aucun template dans le répertoire des templates.",
  },
  "devices.ranking": {
    en: "Ranked by how many of this device's inputs each template can show.",
    fr: "Classés selon le nombre de commandes de ce contrôleur que chaque template peut afficher.",
  },
  "devices.keysUsed": { en: "{score}/{total} keys used", fr: "{score}/{total} clés utilisées" },
  "devices.missingInputs": {
    en: " · {n} input(s) with no key",
    fr: " · {n} commande(s) sans clé",
  },
  "devices.buttonsBadge": { en: "B:{n}", fr: "B:{n}" },
  "devices.axesBadge": { en: "A:{n}", fr: "A:{n}" },
  "devices.hatsBadge": { en: "H:{n}", fr: "H:{n}" },
  "devices.parserFailed": { en: "{parser}: {error}", fr: "{parser} : {error}" },
  "devices.templatesUnconfigured": {
    en: "Templates directory is not configured",
    fr: "Le répertoire des templates n'est pas configuré",
  },
  "devices.templatesReadFailed": {
    en: "Cannot read templates: {error}",
    fr: "Lecture des templates impossible : {error}",
  },

  // ----------------------------------------------------------- generate tab
  "generate.export": { en: "Export", fr: "Exporter" },
  "generate.csvOnly": { en: "CSV only", fr: "CSV seulement" },
  "generate.csvOnlyTitle": {
    en: "Export the CSV only, without generating any diagram",
    fr: "Exporter uniquement le CSV, sans générer de diagramme",
  },
  "generate.run": { en: "Generate", fr: "Générer" },
  "generate.cancel": { en: "Cancel", fr: "Annuler" },
  "generate.targetEverything": { en: "Everything configured", fr: "Tout ce qui est configuré" },
  "generate.notConfigured": {
    en: "Templates and output directories must be set in the Configuration tab before exporting.",
    fr: "Les répertoires des templates et de sortie doivent être renseignés dans l'onglet Configuration avant d'exporter.",
  },
  "generate.statDevices": { en: "devices", fr: "contrôleurs" },
  "generate.statBindings": { en: "bindings", fr: "bindings" },
  "generate.statElapsed": { en: "elapsed", fr: "durée" },
  "generate.simulatorLine": {
    en: "{simulator}: {devices} device(s), {bindings} binding(s)",
    fr: "{simulator} : {devices} contrôleur(s), {bindings} binding(s)",
  },
  "generate.validationTitle": {
    en: "{n} binding(s) have no matching key in their template",
    fr: "{n} binding(s) sans clé correspondante dans leur template",
  },
  "generate.colDevice": { en: "Device", fr: "Contrôleur" },
  "generate.colSimulator": { en: "Simulator", fr: "Simulateur" },
  "generate.colInput": { en: "Input", fr: "Commande" },
  "generate.colAction": { en: "Action", fr: "Action" },
  "generate.colMissingKey": { en: "Missing key", fr: "Clé manquante" },

  // ----------------------------------------------------------- diagrams tab
  "diagrams.openFolder": { en: "Open folder", fr: "Ouvrir le dossier" },
  "diagrams.regenerate": { en: "Regenerate from CSV", fr: "Régénérer depuis le CSV" },
  "diagrams.refresh": { en: "Refresh", fr: "Actualiser" },
  "diagrams.loading": { en: "Reading the output directory…", fr: "Lecture du répertoire de sortie…" },
  "diagrams.summary": {
    en: "{total} diagram(s) in {path}",
    fr: "{total} diagramme(s) dans {path}",
  },
  "diagrams.groupCount": { en: "{n} diagram(s)", fr: "{n} diagramme(s)" },
  "diagrams.noPNG": { en: " · no PNG", fr: " · pas de PNG" },
  "diagrams.svgNote": {
    en: "{name} — SVG preview, labels render correctly in draw.io",
    fr: "{name} — aperçu SVG, les libellés s'affichent correctement dans draw.io",
  },
  "diagrams.outputGroup": { en: "Output directory", fr: "Répertoire de sortie" },
  "diagrams.outputUnconfigured": {
    en: "Output directory is not configured",
    fr: "Le répertoire de sortie n'est pas configuré",
  },
  "diagrams.outputReadFailed": {
    en: "Cannot read the output directory: {error}",
    fr: "Lecture du répertoire de sortie impossible : {error}",
  },
  "diagrams.noneYet": {
    en: "No diagram yet: run an export from the Generate tab.",
    fr: "Aucun diagramme : lancez un export depuis l'onglet Générer.",
  },

  // --------------------------------------------------------------- tips tab
  "tips.startTitle": { en: "Getting started", fr: "Prise en main" },
  "tips.startIntro": {
    en: "SimDiag reads the controller configuration of your simulators and draws a diagram of every controller. Four tabs, in this order:",
    fr: "SimDiag lit la configuration des contrôleurs de vos simulateurs et dessine un diagramme par contrôleur. Quatre onglets, dans cet ordre :",
  },
  "tips.step1": {
    en: "say where your simulators are installed, and which tools you use alongside them.",
    fr: "indiquez où sont installés vos simulateurs, et quels outils vous utilisez avec eux.",
  },
  "tips.step2": {
    en: "pair each controller with the SVG template that draws it.",
    fr: "associez chaque contrôleur au template SVG qui le représente.",
  },
  "tips.step3": {
    en: "run the export, on everything or on a single DCS module.",
    fr: "lancez l'export, sur tout ou sur un seul module DCS.",
  },
  "tips.step4": {
    en: "look at what came out, and open the output folder.",
    fr: "regardez le résultat, et ouvrez le répertoire de sortie.",
  },
  "tips.cliTitle": { en: "Generating without the interface", fr: "Générer sans l'interface" },
  "tips.cliIntro": {
    en: "Once the configuration is done, nothing here has to be clicked again: this command regenerates every diagram from a terminal opened in the folder holding the configuration.",
    fr: "Une fois la configuration faite, plus rien n'a besoin d'être cliqué ici : cette commande régénère tous les diagrammes depuis un terminal ouvert dans le dossier de la configuration.",
  },
  "tips.batIntro": {
    en: "To avoid typing it, SimDiag can write a shortcut that carries the command and the right folder:",
    fr: "Pour ne pas avoir à la taper, SimDiag peut écrire un raccourci qui porte la commande et le bon dossier :",
  },
  "tips.createBat": { en: "Create run_simdiag_batch.bat", fr: "Créer run_simdiag_batch.bat" },
  "tips.batAfter": {
    en: "Double-click the file that gets created to regenerate every diagram. The window stays open at the end so the report can be read.",
    fr: "Double-cliquez sur le fichier créé pour régénérer tous les diagrammes. La fenêtre reste ouverte à la fin, le temps de lire le compte rendu.",
  },
  "tips.linksTitle": { en: "Useful links", fr: "Liens utiles" },
  "tips.linkDrawio": {
    en: "converts the diagrams to PNG, and edits the SVG templates.",
    fr: "convertit les diagrammes en PNG, et permet de modifier les templates SVG.",
  },
  "tips.linkOpenKneeboard": {
    en: "shows the diagrams in game, in VR or on screen.",
    fr: "affiche les diagrammes en jeu, en VR ou à l'écran.",
  },

  // -------------------------------------------------------------- about tab
  "about.tagline": {
    en: "Controller diagram generator for flight simulators.",
    fr: "Générateur de diagrammes de contrôleurs pour simulateurs de vol.",
  },
  "about.updateTitle": { en: "Update", fr: "Mise à jour" },
  "about.checking": { en: "Checking for updates…", fr: "Recherche de mises à jour…" },
  "about.upToDate": {
    en: "You are running the latest version ({version}).",
    fr: "Vous utilisez la dernière version ({version}).",
  },
  "about.available": {
    en: "Version {version} is available.",
    fr: "La version {version} est disponible.",
  },
  "about.published": { en: "published {date}", fr: "publiée le {date}" },
  "about.install": { en: "Install", fr: "Installer" },
  "about.checkNow": { en: "Check for updates", fr: "Vérifier les mises à jour" },
  "about.checkFailed": {
    en: "Could not reach GitHub to check for updates.",
    fr: "GitHub n'a pas pu être contacté pour vérifier les mises à jour.",
  },
  "about.developmentBuild": {
    en: "This is a development build ({version}), which no release can be compared against.",
    fr: "Ceci est un build de développement ({version}), qu'aucune release ne permet de comparer.",
  },
  "about.notesTitle": { en: "Release notes", fr: "Notes de version" },
  "about.seeRelease": { en: "See the release on GitHub", fr: "Voir la release sur GitHub" },
  "about.installing": { en: "Installing…", fr: "Installation en cours…" },
  "about.installed": {
    en: "Version {version} installed. SimDiag must restart to use it.",
    fr: "Version {version} installée. SimDiag doit redémarrer pour l'utiliser.",
  },
  "about.restart": { en: "Restart now", fr: "Redémarrer maintenant" },
  "about.installFailed": { en: "The update failed: {error}", fr: "La mise à jour a échoué : {error}" },
  "about.installCancelled": { en: "Update cancelled.", fr: "Mise à jour annulée." },
  "about.linksTitle": { en: "Links", fr: "Liens" },
  "about.linkRepository": { en: "the source code", fr: "le code source" },
  "about.linkReleases": { en: "every published version", fr: "toutes les versions publiées" },
  "about.linkIssues": { en: "report a problem", fr: "signaler un problème" },
  "about.licenseTitle": { en: "Licence", fr: "Licence" },

  // --------------------------------------------------------------- preview
  "preview.zoomIn": { en: "Zoom in", fr: "Zoom avant" },
  "preview.zoomOut": { en: "Zoom out", fr: "Zoom arrière" },
  "preview.fit": { en: "Fit", fr: "Ajuster" },
  "preview.close": { en: "Close", fr: "Fermer" },

  // -------------------------------------------------- configuration picker
  "picker.recent": { en: "Recent", fr: "Récents" },
  "picker.missing": { en: "missing", fr: "introuvable" },
  "picker.open": { en: "Open…", fr: "Ouvrir…" },
  "picker.new": { en: "New…", fr: "Nouveau…" },
  "picker.reload": { en: "Reload from disk", fr: "Recharger depuis le disque" },

  // --------------------------------------------------------------- confirm
  "confirm.discard": {
    en: "The configuration has unsaved changes. Changing file discards them.",
    fr: "La configuration contient des modifications non enregistrées. Changer de fichier les abandonne.",
  },
  "confirm.cancel": { en: "Cancel", fr: "Annuler" },
  "confirm.continue": { en: "Discard and continue", fr: "Abandonner et continuer" },
  "confirm.batOverwrite": {
    en: "{path} already exists and was not written by SimDiag. Replace it?",
    fr: "{path} existe déjà et n'a pas été écrit par SimDiag. Le remplacer ?",
  },
  "confirm.overwrite": { en: "Replace the file", fr: "Remplacer le fichier" },

  // -------------------------------------------------------- status messages
  "msg.ready": { en: "Ready", fr: "Prêt" },
  "msg.configSaved": { en: "Configuration saved", fr: "Configuration enregistrée" },
  "msg.configLoadFailed": {
    en: "Could not load the configuration: {error}",
    fr: "Chargement de la configuration impossible : {error}",
  },
  "msg.saveFailed": { en: "Could not save: {error}", fr: "Enregistrement impossible : {error}" },
  "msg.backendUnreachable": {
    en: "Cannot reach the SimDiag backend: {error}",
    fr: "SimDiag ne répond pas : {error}",
  },
  "msg.pickerFailed": {
    en: "Could not open the file picker: {error}",
    fr: "Ouverture du sélecteur de fichier impossible : {error}",
  },
  "msg.configSwitched": { en: "Configuration: {path}", fr: "Configuration : {path}" },
  "msg.configSwitchFailed": {
    en: "Could not change configuration: {error}",
    fr: "Changement de configuration impossible : {error}",
  },
  "msg.targetDetected": {
    en: "{n} controller(s) matched to a TARGET device number",
    fr: "{n} contrôleur(s) associé(s) à un numéro de contrôleur TARGET",
  },
  "msg.targetDetectedNone": {
    en: "No controller name matched a TARGET device number",
    fr: "Aucun nom de contrôleur ne correspond à un numéro TARGET",
  },
  "msg.targetDetectFailed": {
    en: "Detection failed: {error}",
    fr: "Détection impossible : {error}",
  },
  "msg.deviceUpdateFailed": {
    en: "Could not update the device: {error}",
    fr: "Mise à jour du contrôleur impossible : {error}",
  },
  "msg.scanning": { en: "Scanning simulator configurations…", fr: "Analyse des configurations…" },
  "msg.scanComplete": { en: "Scan complete", fr: "Analyse terminée" },
  "msg.scanFailed": { en: "Scan failed: {error}", fr: "Analyse échouée : {error}" },
  "msg.diagramsReadFailed": {
    en: "Could not read the output directory: {error}",
    fr: "Lecture du répertoire de sortie impossible : {error}",
  },
  "msg.folderOpenFailed": {
    en: "Could not open the folder: {error}",
    fr: "Ouverture du dossier impossible : {error}",
  },
  "msg.regenerating": { en: "Regenerating diagrams…", fr: "Régénération des diagrammes…" },
  "msg.regenerationFailed": {
    en: "Regeneration failed: {error}",
    fr: "Régénération échouée : {error}",
  },
  "msg.targetsFailed": {
    en: "Could not read the export targets: {error}",
    fr: "Lecture des cibles d'export impossible : {error}",
  },
  "msg.exportRunning": { en: "Export running…", fr: "Export en cours…" },
  "msg.exportFailed": { en: "Export failed: {error}", fr: "Export échoué : {error}" },
  "msg.exportCancelled": { en: "Export cancelled", fr: "Export annulé" },
  "msg.exportComplete": { en: "Export complete", fr: "Export terminé" },
  "msg.cancelling": { en: "Cancelling…", fr: "Annulation…" },
  "msg.cancelFailed": { en: "Could not cancel: {error}", fr: "Annulation impossible : {error}" },
  "msg.templatesInstalled": {
    en: "{n} template(s) installed in {path}",
    fr: "{n} template(s) installé(s) dans {path}",
  },
  "msg.templatesInstallFailed": {
    en: "Could not install the templates: {error}",
    fr: "Installation des templates impossible : {error}",
  },
  "msg.batCreated": { en: "Created {path}", fr: "{path} créé" },
  "msg.batFailed": {
    en: "Could not create the file: {error}",
    fr: "Création du fichier impossible : {error}",
  },
  "msg.restartFailed": {
    en: "Could not restart SimDiag: {error}",
    fr: "Redémarrage de SimDiag impossible : {error}",
  },
  "msg.linkFailed": {
    en: "Could not open the page: {error}",
    fr: "Ouverture de la page impossible : {error}",
  },
  "msg.languageFailed": {
    en: "Could not save the language: {error}",
    fr: "Enregistrement de la langue impossible : {error}",
  },

  // ------------------------------------------- refusals reported by the API
  //
  // These always arrive inside a sentence the page supplies ("Could not change
  // configuration: …"), so they start in lower case and carry no colon of their
  // own. Two colons in a row read badly, in French especially.
  "error.exportRunning": {
    en: "an export is running, wait for it to finish or cancel it",
    fr: "un export est en cours, attendez sa fin ou annulez-le",
  },
  "error.noConfigFile": {
    en: "no configuration file at {path}",
    fr: "aucun fichier de configuration à {path}",
  },
  "error.folderMissing": {
    en: "that folder does not exist yet: {path}",
    fr: "ce dossier n'existe pas encore : {path}",
  },
  "error.noCSVToRegenerate": {
    en: "no export.csv to regenerate from, run an export first",
    fr: "aucun export.csv à régénérer, lancez d'abord un export",
  },
  "error.batchScriptFailed": {
    en: "the file could not be written ({error})",
    fr: "le fichier n'a pas pu être écrit ({error})",
  },
  "error.targetDetectFailed": {
    en: "{error}",
    fr: "{error}",
  },
  "error.updateCheckFailed": {
    en: "GitHub could not be reached ({error})",
    fr: "GitHub n'a pas pu être contacté ({error})",
  },
  "error.updateRunning": {
    en: "an update is already being installed",
    fr: "une mise à jour est déjà en cours d'installation",
  },
  "error.alreadyCurrent": {
    en: "this version is already the latest one",
    fr: "cette version est déjà la plus récente",
  },
  "error.nothingInstalled": {
    en: "nothing has been installed, so there is nothing to restart into",
    fr: "rien n'a été installé, il n'y a donc rien vers quoi redémarrer",
  },
  "error.restartFailed": {
    en: "SimDiag could not be restarted ({error})",
    fr: "SimDiag n'a pas pu être redémarré ({error})",
  },
};

// currentLanguage is the language everything renders in. It is set once the
// session is known, before the first panel is drawn.
export let currentLanguage = "en";

// supportedLanguage narrows anything (a stored preference, navigator.language)
// to a language the catalogue actually has.
function supportedLanguage(value) {
  return String(value || "").toLowerCase().startsWith("fr") ? "fr" : "en";
}

export function setCurrentLanguage(lang) {
  currentLanguage = supportedLanguage(lang);
  document.documentElement.lang = currentLanguage;
}

// t renders a message in the current language, substituting {name} placeholders.
// An unknown key returns itself, which makes the mistake visible on screen
// rather than showing an empty label (gui/i18n_test.go catches it first).
export function t(key, params) {
  const entry = MESSAGES[key];
  if (!entry) return key;

  const text = entry[currentLanguage] ?? entry.en;
  if (!params) return text;

  return text.replace(/\{(\w+)\}/g, (whole, name) =>
    Object.prototype.hasOwnProperty.call(params, name) ? String(params[name]) : whole,
  );
}

// tMessage renders a {code, args} message built by Go (see gui/message.go).
export function tMessage(message) {
  if (!message || !message.code) return "";
  return t(message.code, message.args);
}
