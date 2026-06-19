const article = document.getElementById("article");
const dialog = document.getElementById("annotateDialog");

function getAnnotation() {
  const sel = window.getSelection();
  if (!sel || sel.isCollapsed || !sel.rangeCount) return;

  const range = sel.getRangeAt(0);

  if (!article.contains(range.commonAncestorContainer)) return;

  const pre = range.cloneRange();
  pre.selectNodeContents(article);
  pre.setEnd(range.startContainer, range.startOffset);

  const start = pre.toString().length;
  const end = start + range.toString().length;
  const selection = range.toString();

  console.log(start, end, selection);

  setFields(start, end, selection);
}

function resetState() {
  setFields("", "", "");
  window.getSelection()?.removeAllRanges();
  dialog.close();
}

function closeDialog(e) {
  if (!dialog.open) return;
  if (dialog.contains(e.target)) return;
  resetState();
}

function setFields(start, end, selection) {
  document.getElementById("annotation-start").value = start;
  document.getElementById("annotation-end").value = end;
  document.getElementById("annotation-selection").value = selection;
  document.getElementById("annotation-selection-preview").innerText = selection;
  // always blank
  document.getElementById("annotation-note").value = "";
}
