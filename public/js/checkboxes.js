document.addEventListener("DOMContentLoaded", () => {
  const allCheckbox = document.getElementById("select-all");
  const checkboxes = document.querySelectorAll("input[name='code-checkbox']");
  const codesInput = document.getElementById("codes");

  // Update the state of the "select-all" checkbox based on individual checkboxes
  function updateSelectAllCheckbox() {
    allCheckbox.checked = Array.from(checkboxes).every(
      (checkbox) => checkbox.checked
    );
    codesInput.value = Array.from(checkboxes)
      .filter((checkbox) => checkbox.checked)
      .map((checkbox) => checkbox.value)
      .join(",");
  }

  // Set up event listeners for individual checkboxes
  checkboxes.forEach((checkbox) => {
    checkbox.addEventListener("change", () => {
      updateSelectAllCheckbox();
    });
  });

  // Event listener for the "select-all" checkbox
  allCheckbox.addEventListener("change", (e) => {
    const isChecked = e.target.checked;
    checkboxes.forEach((checkbox) => {
      checkbox.checked = isChecked;
    });
  });

  // Initial check to synchronize the select-all checkbox
  updateSelectAllCheckbox();
});
