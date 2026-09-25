const textarea = document.querySelector("#editor");
const paragraphs = document.querySelectorAll("#article p");

const GROUP_SIZE = 3;

textarea.addEventListener("focus", updateHighlights);
textarea.addEventListener("input", updateHighlights);

function updateHighlights() {
  const text = textarea.value;

  // Split paragraphs using double newlines
  const parts = text.split(/\n\s*\n/);

  // Find the last paragraph containing actual text
  const activeIndex = parts.findLastIndex(
    (paragraph) => paragraph.trim().length > 0,
  );

  // Number of paragraphs actually written
  const writtenCount = parts.filter(
    (paragraph) => paragraph.trim().length > 0,
  ).length;

  paragraphs.forEach((paragraph, index) => {
    // Remove only our custom classes
    paragraph.classList.remove("active", "completed", "summary");

    // Find the group this paragraph belongs to
    const groupStart = Math.floor(index / GROUP_SIZE) * GROUP_SIZE;

    const groupEnd = Math.min(
      groupStart + GROUP_SIZE - 1,
      paragraphs.length - 1,
    );

    // How many paragraphs are required to complete
    // this group? For example, 3, 2, or 1.
    const groupSize = groupEnd - groupStart + 1;

    // Number of written paragraphs in this group
    const writtenInGroup = Math.max(
      0,
      Math.min(writtenCount - groupStart, groupSize),
    );

    // Group is complete only when every paragraph
    // in that group has been written
    const groupComplete = writtenInGroup === groupSize && groupSize > 0;

    // Is this group before the current group?
    const activeGroup =
      activeIndex === -1 ? 0 : Math.floor(activeIndex / GROUP_SIZE);

    const groupIndex = Math.floor(index / GROUP_SIZE);

    if (groupComplete && groupIndex <= activeGroup) {
      // Completed group becomes summary
      paragraph.classList.add("summary");
    } else if (groupIndex === activeGroup) {
      // Current group: highlight completed paragraphs
      // and the paragraph currently being written
      if (index < activeIndex) {
        paragraph.classList.add("completed");
      } else if (index === activeIndex) {
        paragraph.classList.add("active");
      }
    } else if (groupIndex < activeGroup) {
      // Earlier groups remain completed
      paragraph.classList.add("summary");
    }
  });
}
