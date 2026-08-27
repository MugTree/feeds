window.addEventListener("DOMContentLoaded", (e) => {
  feedsBalanceArticleLayout();
});

// ensure the the layout looks good by setting the heights of the righthand column items to the
// heights of the left
function feedsBalanceArticleLayout() {
  console.log("feedsBalanceArticleLayout()");
  document.querySelectorAll("[data-paragraph-id]").forEach((d, i) => {
    const paragraphID = d.getAttribute("data-paragraph-id");
    const paragraphHeight = d.offsetHeight;

    const relatedNote = document.querySelector(
      "[data-note-id='" + paragraphID + "']",
    );

    //  check here to see if right hand content is already taller than the left
    if (relatedNote.offsetHeight < paragraphHeight) {
      relatedNote.style.height = paragraphHeight + "px";
    }
  });
}
