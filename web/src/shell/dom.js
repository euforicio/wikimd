export function getPageRegion() {
  return document.getElementById("page-region");
}

export function getScrollContainer() {
  return document.getElementById("page-scroll");
}

export function currentPath() {
  const region = getPageRegion();
  return region ? region.dataset.currentPath || "" : "";
}

export function setCurrentPath(path) {
  const region = getPageRegion();
  if (region) {
    region.dataset.currentPath = path || "";
  }
}

export function highlightActive(path) {
  const items = document.querySelectorAll("[data-tree-path]");
  const normalised = (path || "").toLowerCase();
  items.forEach((item) => {
    const matches = item.getAttribute("data-tree-path")?.toLowerCase() === normalised;
    item.classList.toggle("tree-link-active", !!matches);
  });
}
