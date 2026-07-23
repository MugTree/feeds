window.addEventListener("DOMContentLoaded", (e) => {
  document.querySelectorAll("[data-block-id]").forEach((e, i) => {
    e.addEventListener("click", (e) => {
      console.log("e :>> ", e);
    });
  });
});
