// The Tips tab. Its prose is static markup in index.html; only the batch-file
// button needs code.
import { setStatus, postJSON, askConfirm } from "./core.js";
import { t } from "./i18n.js";

// createBatchScript writes the .bat that runs `simdiag.exe -b` on this
// configuration, and Go reveals it in the file manager: the point is that the
// user double-clicks it afterwards.
//
// A file already there that SimDiag did not write comes back as an answer, not
// as an error. The name has been in circulation long before this button, and
// someone's own script is not ours to replace unasked.
export async function createBatchScript() {
  const button = document.getElementById("btn-create-bat");
  button.disabled = true;

  try {
    let result = await postJSON("/api/tips/batch-file", {});

    if (result.exists && !result.created) {
      if (!(await askConfirm(t("confirm.batOverwrite", { path: result.path }), t("confirm.overwrite")))) {
        return;
      }
      result = await postJSON("/api/tips/batch-file", { overwrite: true });
    }

    setStatus("msg.batCreated", { path: result.path });
  } catch (err) {
    setStatus("msg.batFailed", { error: err.message }, true);
  } finally {
    button.disabled = false;
  }
}
