const BASE = document.getElementById("base");

function getSelectionRangeData() {
  function getNodePath(node) {
    const path = [];

    while (node && node !== BASE) {
      const parent = node.parentNode;
      if (!parent) return null;

      // reads easier that doing push and then reversing on return
      path.unshift(Array.prototype.indexOf.call(parent.childNodes, node));
      node = parent;
    }
    return path;
  }

  const sel = window.getSelection();

  if (!sel || sel.rangeCount == 0 || sel.isCollapsed) return null;

  // how can you have more than one selected range??
  // does that even make sense
  const range = sel.getRangeAt(0);
  const s = {
    selection: sel.toString(),
    start: {
      path: getNodePath(range.startContainer),
      offset: range.startOffset,
    },
    end: {
      path: getNodePath(range.endContainer),
      offset: range.endOffset,
    },
  };

  return s;
}
