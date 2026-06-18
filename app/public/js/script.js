const article = document.getElementById("article");
const dialog = document.getElementById("annotateDialog");

function getAnnotationData() {
  const sel = window.getSelection();
  if (!sel || sel.isCollapsed || !sel.rangeCount) return;

  const range = sel.getRangeAt(0);

  if (!article.contains(range.commonAncestorContainer)) return;

  const pre = range.cloneRange();
  pre.selectNodeContents(article);
  pre.setEnd(range.startContainer, range.startOffset);

  const start = pre.toString().length;
  const end = start + range.toString().length;

  const data = {
    start: start,
    end: end,
    selection: range.toString(),
    note: "",
  };

  return data;
}

function resetState(selectionData) {
  selectionData = null;
  document.getElementById("note").value = "";
  window.getSelection()?.removeAllRanges();
  dialog.close();
}

function closeDialog(e) {
  if (!dialog.open) return;
  if (dialog.contains(e.target)) return;
  resetState();
}
