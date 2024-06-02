/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["*.templ", "**/*.templ"],
  theme: {
    extend: {},
  },
  plugins: [require("@tailwindcss/forms"), require("@tailwindcss/typography")],
};
