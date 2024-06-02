// This script is used to toggle the loader when a form is submitted.
document.addEventListener("DOMContentLoaded", () => {
  const forms = document.querySelectorAll("form");
  forms.forEach((form) => {
    form.addEventListener("submit", (e) => {
      e.preventDefault();
      const loaders = form.querySelectorAll("[data-loader]");
      loaders.forEach((loader) => loader.classList.toggle("hidden"));
      const buttons = form.querySelectorAll("button[type=submit]");
      buttons.forEach((button) => (button.disabled = true));
      form.submit();
    });
  });
});
