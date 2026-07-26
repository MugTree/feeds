window.addEventListener("DOMContentLoaded", (e) => {
  document.querySelectorAll("[data-block-id]").forEach((d, i) => {
    console.log("d :>> ", d);
    d.addEventListener("click", (e) => {
      console.log("e :>> ", e);
    });
  });
});
