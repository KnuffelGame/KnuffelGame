// ./knuffel-frontend/tailwind.config.js (ES Module Syntax)

/** @type {import('tailwindcss').Config} */
// Entfernt 'module.exports' und verwendet 'export default'
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}", 
  ],
  theme: {
    extend: {},
  },
  plugins: [],
};