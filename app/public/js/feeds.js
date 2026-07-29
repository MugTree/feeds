window.addEventListener("DOMContentLoaded", (e) => {
  feedsBalanceArticleLayout();
});

// ensure the the layout looks good by setting the heights of the righthand column items to the
// heights of the left
function feedsBalanceArticleLayout() {
  console.log("feedsBalanceArticleLayout()");
  document.querySelectorAll("[data-block-id]").forEach((d, i) => {
    const blockID = d.getAttribute("data-block-id");
    const blockHeight = d.offsetHeight;

    const relatedNote = document.querySelector(
      "[data-note-id='" + blockID + "']",
    );

    relatedNote.style.height = blockHeight + "px";
  });
}
