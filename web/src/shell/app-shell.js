import { loadThemePreference } from "../shared/theme";
import { enhanceContent } from "../shared/content";
import { currentPath, getPageRegion, getScrollContainer, highlightActive, setCurrentPath } from "./dom";

function fetchTree() {
  const path = currentPath();
  const url = path ? `/api/tree?current=${encodeURIComponent(path)}` : "/api/tree";
  if (window.htmx) {
    window.htmx.ajax("GET", url, { target: "#nav-tree", swap: "innerHTML" });
  }
}

function fetchPage(path, preserveScroll, pendingScrollRef) {
  if (!window.htmx || !path) {
    return;
  }
  const scroller = getScrollContainer();
  if (preserveScroll && scroller) {
    pendingScrollRef.value = scroller.scrollTop;
  } else {
    pendingScrollRef.value = null;
  }
  window.htmx.ajax("GET", `/api/page/${encodeURIComponent(path)}`, {
    target: "#page-region",
    swap: "innerHTML",
  });
}

function connectEvents(pendingScrollRef) {
  let retryDelay = 1000;
  const source = new EventSource("/events");

  source.onmessage = (event) => {
    if (!event.data) {
      return;
    }
    try {
      const payload = JSON.parse(event.data);
      switch (payload.type) {
        case "treeUpdated":
          fetchTree();
          break;
        case "pageUpdated":
          if (payload.path && payload.path === currentPath()) {
            fetchPage(payload.path, true, pendingScrollRef);
          }
          break;
        case "deleted":
          if (payload.path && payload.path === currentPath()) {
            setCurrentPath("");
            const region = getPageRegion();
            if (region) {
              region.innerHTML = `<div class="rounded-2xl border border-dashed border-red-500/40 bg-red-500/10 p-6 text-sm text-red-200">The document <code>${payload.path}</code> was deleted.</div>`;
            }
          }
          fetchTree();
          break;
        default:
          break;
      }
    } catch (err) {
      console.warn("sse message error", err);
    }
  };

  source.onerror = () => {
    source.close();
    setTimeout(() => connectEvents(pendingScrollRef), retryDelay);
    retryDelay = Math.min(retryDelay * 2, 10000);
  };
}

function showSearch() {
  const panel = document.getElementById("search-results");
  panel?.classList.remove("hidden");
}

function hideSearch() {
  const panel = document.getElementById("search-results");
  if (panel) {
    panel.classList.add("hidden");
    panel.innerHTML = "";
  }
}


function bindPageLifecycle(pendingScrollRef) {
  document.body.addEventListener("pageLoaded", (event) => {
    const detail = event.detail || {};
    setCurrentPath(detail.path || "");
    const region = getPageRegion();
    enhanceContent(region);
    const scroller = getScrollContainer();
    if (pendingScrollRef.value !== null && scroller) {
      scroller.scrollTop = pendingScrollRef.value;
      pendingScrollRef.value = null;
    } else if (scroller) {
      scroller.scrollTop = 0;
    }
    hideSearch();
  });

  document.body.addEventListener("treeUpdated", () => {
    highlightActive(currentPath());
  });

  document.body.addEventListener("searchResults", (event) => {
    const detail = event.detail || {};
    if (!detail.query) {
      hideSearch();
      return;
    }
    showSearch();
  });
}

function bindHtmxEnhancements() {
  if (!window.htmx) {
    return;
  }
  window.htmx.on("htmx:afterSwap", (evt) => {
    const targetId = evt.detail.target?.id;
    if (targetId === "page-region") {
      const region = getPageRegion();
      if (region) {
        enhanceContent(region);
      }
    }
  });
}

function bindShortcuts({ hideSearch }) {
  document.addEventListener("keydown", (event) => {
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
      event.preventDefault();
      const input = document.getElementById("search-query");
      if (input) {
        input.focus();
      }
    } else if (event.key === "Escape") {
      hideSearch();
    }
  });
}

function bindSearchDismiss() {
  document.addEventListener("click", (event) => {
    const panel = document.getElementById("search-results");
    const searchForm = document.getElementById("search-form");
    if (!panel || panel.classList.contains("hidden")) {
      return;
    }
    if (!panel.contains(event.target) && !searchForm?.contains(event.target)) {
      hideSearch();
    }
  });
}

function createToast() {
  let toastTimer = null;
  return {
    show(message) {
      if (!message) {
        return;
      }
      let toast = document.getElementById("wikimd-toast");
      if (!toast) {
        toast = document.createElement("div");
        toast.id = "wikimd-toast";
        toast.className =
          "pointer-events-none fixed bottom-6 right-6 z-50 max-w-sm rounded-2xl border border-slate-800/80 bg-surface/90 px-5 py-3 text-sm text-slate-200 shadow-2xl backdrop-blur transition-all duration-300 ease-out opacity-0 translate-y-4";
        document.body.appendChild(toast);
      }
      toast.textContent = message;
      requestAnimationFrame(() => {
        toast.classList.remove("opacity-0", "translate-y-4");
        toast.classList.add("opacity-100", "translate-y-0");
      });
      if (toastTimer) {
        clearTimeout(toastTimer);
      }
      toastTimer = setTimeout(() => {
        toast.classList.remove("opacity-100", "translate-y-0");
        toast.classList.add("opacity-0", "translate-y-4");
      }, 3200);
    },
  };
}


export function initAppShell() {
  const pendingScrollRef = { value: null };
  const toast = createToast();

  // Load theme FIRST, before any rendering happens
  loadThemePreference();

  bindPageLifecycle(pendingScrollRef);
  bindHtmxEnhancements();
  bindShortcuts({ hideSearch });
  bindSearchDismiss();

  window.addEventListener("load", () => {
    highlightActive(currentPath());
    const region = getPageRegion();
    enhanceContent(region);
    connectEvents(pendingScrollRef);
  });
}
