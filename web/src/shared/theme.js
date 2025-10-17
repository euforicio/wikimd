// Simple theme initialization - always defaults to dark theme
// Custom theming is now handled via CSS files (see THEMING_PLAN.md)

export function loadThemePreference() {
  const root = document.documentElement;
  // Always use dark theme
  root.classList.add("dark");
  root.dataset.theme = "wikimd";
}
