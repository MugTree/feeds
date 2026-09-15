// window.addEventListener("DOMContentLoaded", (e) => {
//   feedsBalanceArticleLayout();
// });

// ensure the the layout looks good by setting the heights of the righthand column items to the
// heights of the left
// function feedsBalanceArticleLayout() {
//   console.log("feedsBalanceArticleLayout()");
//   document.querySelectorAll("[data-paragraph-id]").forEach((d, i) => {
//     const paragraphID = d.getAttribute("data-paragraph-id");
//     const paragraphHeight = d.offsetHeight;

//     const relatedNote = document.querySelector(
//       "[data-comment-id='" + paragraphID + "']",
//     );

//     //  check here to see if right hand content is already taller than the left
//     if (relatedNote.offsetHeight < paragraphHeight) {
//       relatedNote.style.height = paragraphHeight + "px";
//     }
//   });
// }

function equaliseHeights() {
  const nodes = document.querySelectorAll(".paragraphs p");
  const paragraphs = [...nodes].map((n) => n);
  const heights = paragraphs.map((p) => p.getBoundingClientRect().height);

  const noteNodes = document.querySelectorAll("#notes p");
  const notes = [...noteNodes].map((n) => n);

  console.log("notes :>> ", notes);

  for (let i = 0; i < notes.length; i++) {
    notes[i].setAttribute("style", "height: " + heights[i] + "px");
  }
}

function twoConsecutiveNewlines(evt) {
  const userInput = evt.target;
  const position = userInput.selectionStart;
  const text = userInput.value;
  if (text[position - 1] === "\n" && text[position - 2] === "\n") {
    console.log("New paragraph created!");
    return true;
  }
  return false;
}
