let annotations;
const BASE = document.getElementById("base");

async function loadData() {
  const url = "http://localhost:8081/fake-annotation-test/data";

  try {
    const res = await fetch(url);
    if (!res.ok) {
      throw new Error(`Response status: ${res.status}`);
    }

    annotations = await res.json();
    console.log("annotations :>> ", annotations);

    annotations.forEach((a) => {
      const range = buildRange(a);
      highlightRange(range, a.id);
    });
  } catch (error) {
    console.log("error :>> ", error);
  }
}

function buildRange(annotation) {
  const range = document.createRange();

  const startNode = getNodeFromPath(annotation.start_data.path);
  const endNode = getNodeFromPath(annotation.end_data.path);

  console.log("startNode :>> ", startNode);
  console.log("endNode :>> ", endNode);

  range.setStart(startNode, annotation.start_data.offset);
  range.setEnd(endNode, annotation.end_data.offset);

  return range;
}

function getNodePath(node) {
  const path = [];

  while (node && node !== BASE) {
    const parent = node.parentNode;
    if (!parent) return null;

    path.unshift(Array.prototype.indexOf.call(parent.childNodes, node));
    node = parent;
  }

  //console.log("path is :>> ", path);

  return path;
}

function getNodeFromPath(path) {
  let node = BASE;

  for (const index of path) {
    if (index < 0 || index >= node.childNodes.length) {
      return null;
    }

    node = node.childNodes[index];
  }

  return node;
}

function highlightRange(range, annotationId) {
  const contents = range.cloneContents();

  const wrapper = document.createElement("span");
  wrapper.className = "highlight";
  wrapper.dataset.id = annotationId;

  wrapper.appendChild(contents);

  range.deleteContents();
  range.insertNode(wrapper);
}

function getSelectionRangeData() {
  const sel = window.getSelection();

  if (!sel || sel.rangeCount == 0 || sel.isCollapsed) return null;

  // how can you have more than one selected range??
  const range = sel.getRangeAt(0);

  // get the nodes where the selection starts and ends
  const startNode = range.startContainer;
  const endNode = range.endContainer;

  const s = {
    selection: sel.toString(),
    start: {
      path: getNodePath(startNode),
      offset: range.startOffset,
    },
    end: {
      path: getNodePath(endNode),
      offset: range.endOffset,
    },
  };

  console.log("range data :>> ", s);

  return s;
}

loadData();
