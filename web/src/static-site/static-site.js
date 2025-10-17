import { loadThemePreference } from "../shared/theme";
import { enhanceContent, reRenderMermaid } from "../shared/content";

// Expose reRenderMermaid globally (if needed for future use)
window.reRenderMermaid = reRenderMermaid;

function highlightActive() {
  const active = (document.body.dataset.page || "").toLowerCase();
  const items = document.querySelectorAll("[data-tree-path]");
  items.forEach((item) => {
    const candidate = (item.getAttribute("data-tree-path") || "").toLowerCase();
    item.classList.toggle("tree-link-active", candidate === active);
  });
}

export function initStaticSite() {
  // Load theme FIRST, before DOM loads to prevent flash
  loadThemePreference();

  document.addEventListener("DOMContentLoaded", () => {
    highlightActive();
    const region = document.getElementById("page-region");
    enhanceContent(region);
  });
}
