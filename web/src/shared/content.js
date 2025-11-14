const MERMAID_OVERLAY_VISIBLE_CLASS = "mermaid-overlay-open";
const OVERLAY_MIN_SCALE = 0.25;
const OVERLAY_MAX_SCALE = 5;
const OVERLAY_WHEEL_SENSITIVITY = 0.002;

let mermaidOverlayRoot = null;
let mermaidOverlayContent = null;
let mermaidOverlayViewport = null;
let mermaidOverlayCanvas = null;
let mermaidOverlayKeyListenerAttached = false;
let mermaidOverlayPointerHandlersBound = false;

const mermaidOverlayPointerState = new Map();
const mermaidOverlayTransform = { scale: 1, x: 0, y: 0 };
let lastPinchDistance = null;

export function enhanceContent(element) {
  if (!element) {
    return;
  }

  normaliseInternalLinks(element);
  addCopyButtonsToCodeBlocks(element);
  renderMermaid(element);
  enhanceD2(element);
}

function enhanceD2(element) {
  if (!element) {
    return;
  }

  const d2Blocks = element.querySelectorAll(".d2-block");
  if (!d2Blocks || d2Blocks.length === 0) {
    return;
  }

  d2Blocks.forEach((block) => {
    if (!block || block.dataset.d2Enhanced === "true") {
      return;
    }

    block.dataset.d2Enhanced = "true";
    block.style.position = block.style.position || "relative";

    if (!block.querySelector(".diagram-expand-button")) {
      const button = createDiagramExpandButton(() => openD2Overlay(block));
      block.appendChild(button);
    }
  });
}

async function renderMermaid(element) {
  if (!window.mermaid || !element) {
    return;
  }

  try {
    const mermaidElements = element.querySelectorAll(".mermaid");

    if (mermaidElements.length === 0) {
      return;
    }

    // Theme variables for light mode to match GitHub/Gist style
    const lightThemeVars = {
      primaryColor: "#ffffff",
      primaryTextColor: "#24292e",
      primaryBorderColor: "#e1e4e8",
      lineColor: "#d1d5da",
      secondaryColor: "#f6f8fa",
      tertiaryColor: "#f6f8fa",
      fontFamily: "-apple-system,BlinkMacSystemFont,Segoe UI,Helvetica,Arial,sans-serif,Apple Color Emoji,Segoe UI Emoji",
      fontSize: "14px",
    };

    // A simplified dark theme
    const darkThemeVars = {
      primaryColor: "#1c2333",
      primaryTextColor: "#f8fafc",
      primaryBorderColor: "#30384d",
      lineColor: "#3d475f",
      secondaryColor: "#141a29",
      tertiaryColor: "#141a29",
      fontFamily: "Inter, ui-sans-serif, system-ui",
      fontSize: "14px",
    };

    // Detect current theme
    const isDark = document.documentElement.classList.contains("dark");

    // Initialize mermaid with appropriate theme
    window.mermaid.initialize({
      startOnLoad: false,
      theme: isDark ? "dark" : "base",
      themeVariables: isDark ? darkThemeVars : lightThemeVars,
    });

    // Process each mermaid element
    for (const el of mermaidElements) {
      ensureMermaidContainer(el);

      // Skip if already processed
      if (el.dataset.mermaidProcessed === "true") {
        continue;
      }

      const mermaidCode = el.textContent.trim();
      if (!mermaidCode) {
        continue;
      }

      // Store original code in data attribute for re-rendering
      el.dataset.mermaidSource = mermaidCode;

      try {
        await window.mermaid.run({ nodes: [el] });
        el.dataset.mermaidProcessed = "true";
      } catch (err) {
        console.error("mermaid render failed:", err);
        el.innerHTML = `<div style="color: red; border: 1px solid red; padding: 10px; border-radius: 4px;">
          <strong>Mermaid Error:</strong><br>
          ${err.message || "Failed to render diagram"}
        </div>`;
      }
    }
  } catch (err) {
    console.error("mermaid render failed:", err);
  }
}

function ensureMermaidContainer(element) {
  if (!element) {
    return null;
  }

  let wrapper = element.parentElement;
  const wrapperClass = "mermaid-block";

  if (!wrapper || !wrapper.classList.contains(wrapperClass)) {
    wrapper = document.createElement("div");
    wrapper.className = wrapperClass;
    if (element.parentNode) {
      element.parentNode.insertBefore(wrapper, element);
      wrapper.appendChild(element);
    }
  }

  if (wrapper && !wrapper.querySelector(".mermaid-expand-button")) {
    const button = createMermaidExpandButton(element);
    wrapper.appendChild(button);
  }

  return wrapper;
}

function createMermaidExpandButton(targetElement) {
  return createDiagramExpandButton(() => openMermaidOverlay(targetElement));
}

function createDiagramExpandButton(onTrigger) {
  const button = document.createElement("button");
  button.type = "button";
  button.className = "mermaid-expand-button diagram-expand-button";
  button.setAttribute("aria-label", "Expand diagram");
  button.setAttribute("title", "Expand diagram");
  button.innerHTML = `
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <polyline points="14 3 21 3 21 10"></polyline>
      <line x1="21" y1="3" x2="14" y2="10"></line>
      <polyline points="10 21 3 21 3 14"></polyline>
      <line x1="3" y1="21" x2="10" y2="14"></line>
    </svg>
  `;

  button.addEventListener("click", (event) => {
    event.preventDefault();
    event.stopPropagation();
    onTrigger?.(event);
  });

  return button;
}

async function openMermaidOverlay(sourceElement) {
  const { overlay, canvas } = ensureMermaidOverlayElements();
  if (!overlay || !canvas) {
    return;
  }

  canvas.innerHTML = "";
  resetOverlayTransform();

  const source = sourceElement?.dataset?.mermaidSource || sourceElement?.textContent || "";
  if (!source.trim()) {
    const error = document.createElement("div");
    error.className = "mermaid-overlay-error";
    error.textContent = "No diagram content available.";
    canvas.appendChild(error);
    showMermaidOverlay(overlay);
    return;
  }

  const existingSvg = sourceElement.querySelector("svg");
  if (existingSvg) {
    const clonedWrapper = document.createElement("div");
    clonedWrapper.className = "mermaid mermaid-overlay-diagram";
    const clonedSvg = existingSvg.cloneNode(true);
    clonedWrapper.appendChild(clonedSvg);
    canvas.appendChild(clonedWrapper);
    showMermaidOverlay(overlay);
    scheduleOverlayFitToViewport();
    return;
  }

  const overlayDiagram = document.createElement("div");
  overlayDiagram.className = "mermaid mermaid-overlay-diagram";
  overlayDiagram.textContent = source;
  canvas.appendChild(overlayDiagram);

  showMermaidOverlay(overlay);

  try {
    await window.mermaid.run({ nodes: [overlayDiagram] });
    scheduleOverlayFitToViewport();
  } catch (err) {
    overlayDiagram.innerHTML = `<div class="mermaid-overlay-error">${err?.message || "Failed to render diagram"}</div>`;
  }
}

function openD2Overlay(sourceBlock) {
  const { overlay, canvas } = ensureMermaidOverlayElements();
  if (!overlay || !canvas) {
    return;
  }

  canvas.innerHTML = "";
  resetOverlayTransform();

  const svg = sourceBlock?.querySelector("svg");
  if (!svg) {
    const error = document.createElement("div");
    error.className = "mermaid-overlay-error";
    error.textContent = "Diagram SVG is not available.";
    canvas.appendChild(error);
    showMermaidOverlay(overlay);
    return;
  }

  const wrapper = document.createElement("div");
  wrapper.className = "mermaid-overlay-diagram d2-overlay-diagram";
  wrapper.appendChild(svg.cloneNode(true));
  canvas.appendChild(wrapper);

  showMermaidOverlay(overlay);
  scheduleOverlayFitToViewport();
}

function showMermaidOverlay(overlay) {
  overlay.classList.add(MERMAID_OVERLAY_VISIBLE_CLASS);
  overlay.setAttribute("aria-hidden", "false");

  if (!mermaidOverlayKeyListenerAttached) {
    window.addEventListener("keydown", handleOverlayKeydown);
    mermaidOverlayKeyListenerAttached = true;
  }
}

function ensureMermaidOverlayElements() {
  if (mermaidOverlayRoot && mermaidOverlayContent && mermaidOverlayCanvas && mermaidOverlayViewport) {
    return { overlay: mermaidOverlayRoot, content: mermaidOverlayContent, canvas: mermaidOverlayCanvas };
  }

  const overlay = document.createElement("div");
  overlay.id = "mermaid-overlay";
  overlay.className = "mermaid-overlay";
  overlay.setAttribute("aria-hidden", "true");

  const panel = document.createElement("div");
  panel.className = "mermaid-overlay-panel";
  panel.setAttribute("role", "dialog");
  panel.setAttribute("aria-modal", "true");

  const toolbar = document.createElement("div");
  toolbar.className = "mermaid-overlay-toolbar";

  const title = document.createElement("span");
  title.className = "mermaid-overlay-title";
  title.textContent = "Diagram preview";

  const actions = document.createElement("div");
  actions.className = "mermaid-overlay-actions";

  const resetButton = document.createElement("button");
  resetButton.type = "button";
  resetButton.className = "mermaid-overlay-reset";
  resetButton.setAttribute("aria-label", "Reset zoom and position");
  resetButton.innerHTML = `
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <polyline points="1 4 1 10 7 10"></polyline>
      <polyline points="23 20 23 14 17 14"></polyline>
      <path d="M20.49 9A9 9 0 0 0 5.64 5.64L1 10"></path>
      <path d="M3.51 15A9 9 0 0 0 18.36 18.36L23 14"></path>
    </svg>
    <span>Reset</span>
  `;

  const closeButton = document.createElement("button");
  closeButton.type = "button";
  closeButton.className = "mermaid-overlay-close";
  closeButton.setAttribute("aria-label", "Close diagram");
  closeButton.innerHTML = `
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <line x1="18" y1="6" x2="6" y2="18"></line>
      <line x1="6" y1="6" x2="18" y2="18"></line>
    </svg>
  `;

  const hint = document.createElement("p");
  hint.className = "mermaid-overlay-hint";
  hint.textContent = "Scroll or pinch to explore the full diagram.";

  const content = document.createElement("div");
  content.className = "mermaid-overlay-content";

  const viewport = document.createElement("div");
  viewport.className = "mermaid-overlay-viewport";

  const canvas = document.createElement("div");
  canvas.className = "mermaid-overlay-canvas";

  viewport.appendChild(canvas);
  content.appendChild(viewport);

  actions.appendChild(resetButton);
  actions.appendChild(closeButton);

  toolbar.appendChild(title);
  toolbar.appendChild(actions);
  panel.appendChild(toolbar);
  panel.appendChild(hint);
  panel.appendChild(content);
  overlay.appendChild(panel);

  document.body.appendChild(overlay);

  closeButton.addEventListener("click", () => closeMermaidOverlay());
  resetButton.addEventListener("click", () => resetOverlayTransform({ fitToViewport: true }));
  overlay.addEventListener("click", (event) => {
    if (event.target === overlay) {
      closeMermaidOverlay();
    }
  });

  mermaidOverlayRoot = overlay;
  mermaidOverlayContent = content;
  mermaidOverlayViewport = viewport;
  mermaidOverlayCanvas = canvas;

  bindOverlayInteractionHandlers();

  return { overlay, content, canvas };
}

function bindOverlayInteractionHandlers() {
  if (!mermaidOverlayViewport || mermaidOverlayPointerHandlersBound) {
    return;
  }

  mermaidOverlayViewport.addEventListener("wheel", handleOverlayWheel, { passive: false });
  mermaidOverlayViewport.addEventListener("pointerdown", handleOverlayPointerDown);
  mermaidOverlayViewport.addEventListener("pointermove", handleOverlayPointerMove);
  mermaidOverlayViewport.addEventListener("pointerup", handleOverlayPointerUp);
  mermaidOverlayViewport.addEventListener("pointerleave", handleOverlayPointerUp);
  mermaidOverlayViewport.addEventListener("pointercancel", handleOverlayPointerUp);

  mermaidOverlayPointerHandlersBound = true;
}

function handleOverlayWheel(event) {
  if (!mermaidOverlayViewport) {
    return;
  }

  event.preventDefault();

  if (event.ctrlKey || event.metaKey) {
    const rect = mermaidOverlayViewport.getBoundingClientRect();
    const point = {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top,
    };
    const factor = Math.exp(-event.deltaY * OVERLAY_WHEEL_SENSITIVITY);
    zoomOverlay(factor, point);
  } else {
    panOverlay(-event.deltaX, -event.deltaY);
  }
}

function handleOverlayPointerDown(event) {
  if (!mermaidOverlayViewport) {
    return;
  }

  if (event.pointerType === "mouse" && event.button !== 0) {
    return;
  }

  event.preventDefault();
  if (typeof mermaidOverlayViewport.setPointerCapture === "function") {
    try {
      mermaidOverlayViewport.setPointerCapture(event.pointerId);
    } catch {
      // Ignore capture errors on unsupported elements
    }
  }
  mermaidOverlayPointerState.set(event.pointerId, { x: event.clientX, y: event.clientY });

  if (mermaidOverlayPointerState.size === 1) {
    mermaidOverlayViewport.classList.add("is-dragging");
  } else if (mermaidOverlayPointerState.size === 2) {
    lastPinchDistance = getCurrentPinchDistance();
  }
}

function handleOverlayPointerMove(event) {
  if (!mermaidOverlayPointerState.has(event.pointerId)) {
    return;
  }

  event.preventDefault();
  const previous = mermaidOverlayPointerState.get(event.pointerId);
  const current = { x: event.clientX, y: event.clientY };
  mermaidOverlayPointerState.set(event.pointerId, current);

  if (mermaidOverlayPointerState.size === 1 && previous) {
    panOverlay(current.x - previous.x, current.y - previous.y);
    return;
  }

  if (mermaidOverlayPointerState.size >= 2) {
    const distance = getCurrentPinchDistance();
    const center = getCurrentPinchCenter();
    if (lastPinchDistance && distance && center) {
      const factor = distance / lastPinchDistance;
      zoomOverlay(factor, center);
    }
    lastPinchDistance = distance;
  }
}

function handleOverlayPointerUp(event) {
  if (!mermaidOverlayPointerState.has(event.pointerId)) {
    return;
  }

  mermaidOverlayPointerState.delete(event.pointerId);

  if (
    mermaidOverlayViewport &&
    typeof mermaidOverlayViewport.releasePointerCapture === "function"
  ) {
    try {
      mermaidOverlayViewport.releasePointerCapture(event.pointerId);
    } catch {
      // noop - element may not have capture
    }
  }

  if (mermaidOverlayPointerState.size === 0 && mermaidOverlayViewport) {
    mermaidOverlayViewport.classList.remove("is-dragging");
  }

  if (mermaidOverlayPointerState.size < 2) {
    lastPinchDistance = null;
  }
}

function getCurrentPinchDistance() {
  if (mermaidOverlayPointerState.size < 2) {
    return null;
  }

  const [first, second] = Array.from(mermaidOverlayPointerState.values());
  if (!first || !second) {
    return null;
  }

  const dx = first.x - second.x;
  const dy = first.y - second.y;
  return Math.hypot(dx, dy);
}

function getCurrentPinchCenter() {
  if (!mermaidOverlayViewport || mermaidOverlayPointerState.size < 2) {
    return null;
  }

  const [first, second] = Array.from(mermaidOverlayPointerState.values());
  if (!first || !second) {
    return null;
  }

  const rect = mermaidOverlayViewport.getBoundingClientRect();
  return {
    x: (first.x + second.x) / 2 - rect.left,
    y: (first.y + second.y) / 2 - rect.top,
  };
}

function panOverlay(deltaX, deltaY) {
  if (!mermaidOverlayCanvas) {
    return;
  }

  mermaidOverlayTransform.x += deltaX;
  mermaidOverlayTransform.y += deltaY;
  applyOverlayTransform();
}

function zoomOverlay(factor, point) {
  if (!mermaidOverlayCanvas || !mermaidOverlayViewport) {
    return;
  }

  const previousScale = mermaidOverlayTransform.scale;
  const nextScale = clampScale(previousScale * factor);
  if (nextScale === previousScale) {
    return;
  }

  const appliedFactor = nextScale / previousScale;
  const focusPoint = point || {
    x: mermaidOverlayViewport.clientWidth / 2,
    y: mermaidOverlayViewport.clientHeight / 2,
  };

  mermaidOverlayTransform.x = focusPoint.x - appliedFactor * (focusPoint.x - mermaidOverlayTransform.x);
  mermaidOverlayTransform.y = focusPoint.y - appliedFactor * (focusPoint.y - mermaidOverlayTransform.y);
  mermaidOverlayTransform.scale = nextScale;
  applyOverlayTransform();
}

function applyOverlayTransform() {
  if (!mermaidOverlayCanvas) {
    return;
  }

  const { scale, x, y } = mermaidOverlayTransform;
  mermaidOverlayCanvas.style.transform = `matrix(${scale}, 0, 0, ${scale}, ${x}, ${y})`;
}

function resetOverlayTransform(options = {}) {
  const { fitToViewport = false } = options;
  mermaidOverlayTransform.scale = 1;
  mermaidOverlayTransform.x = 0;
  mermaidOverlayTransform.y = 0;
  mermaidOverlayPointerState.clear();
  lastPinchDistance = null;

  if (mermaidOverlayViewport) {
    mermaidOverlayViewport.classList.remove("is-dragging");
  }

  applyOverlayTransform();

  if (fitToViewport) {
    scheduleOverlayFitToViewport();
  }
}

function clampScale(value) {
  return Math.min(Math.max(value, OVERLAY_MIN_SCALE), OVERLAY_MAX_SCALE);
}

function scheduleOverlayFitToViewport() {
  if (!mermaidOverlayViewport) {
    return;
  }

  requestAnimationFrame(() => {
    requestAnimationFrame(() => fitOverlayDiagramToViewport());
  });
}

function fitOverlayDiagramToViewport() {
  if (!mermaidOverlayViewport || !mermaidOverlayCanvas) {
    return;
  }

  const diagramSvg = mermaidOverlayCanvas.querySelector('.mermaid-overlay-diagram svg');
  if (!diagramSvg) {
    return;
  }

  const viewportRect = mermaidOverlayViewport.getBoundingClientRect();
  const diagramRect = diagramSvg.getBoundingClientRect();

  if (
    viewportRect.width === 0 ||
    viewportRect.height === 0 ||
    diagramRect.width === 0 ||
    diagramRect.height === 0
  ) {
    return;
  }

  const paddingFactor = 0.9;
  const scaleToFit = Math.min(
    viewportRect.width / diagramRect.width,
    viewportRect.height / diagramRect.height,
  ) * paddingFactor;

  const targetScale = clampScale(Math.max(scaleToFit, 0.1));

  const offsetX = diagramRect.left - viewportRect.left;
  const offsetY = diagramRect.top - viewportRect.top;
  const centerX = offsetX + diagramRect.width / 2;
  const centerY = offsetY + diagramRect.height / 2;

  mermaidOverlayTransform.scale = targetScale;
  mermaidOverlayTransform.x = viewportRect.width / 2 - centerX * targetScale;
  mermaidOverlayTransform.y = viewportRect.height / 2 - centerY * targetScale;

  applyOverlayTransform();
}

function handleOverlayKeydown(event) {
  if (event.key === "Escape") {
    closeMermaidOverlay();
  }
}

function closeMermaidOverlay() {
  if (!mermaidOverlayRoot) {
    return;
  }

  mermaidOverlayRoot.classList.remove(MERMAID_OVERLAY_VISIBLE_CLASS);
  mermaidOverlayRoot.setAttribute("aria-hidden", "true");

  if (mermaidOverlayCanvas) {
    mermaidOverlayCanvas.innerHTML = "";
  }

  resetOverlayTransform();

  if (mermaidOverlayKeyListenerAttached) {
    window.removeEventListener("keydown", handleOverlayKeydown);
    mermaidOverlayKeyListenerAttached = false;
  }
}

// Re-render all mermaid diagrams when theme changes
export function reRenderMermaid() {
  if (!window.mermaid) {
    return;
  }

  const containers = document.querySelectorAll("[data-mermaid-source]");

  containers.forEach((container) => {
    const source = container.dataset.mermaidSource;
    if (!source) {
      return;
    }

    // Restore original code and reset processing flag
    container.textContent = source;
    container.classList.add("mermaid");
    delete container.dataset.mermaidProcessed;
  });

  // Re-render all diagrams with current theme
  const pageRegion = document.getElementById("page-region");
  if (pageRegion) {
    renderMermaid(pageRegion);
  }
}

function normaliseInternalLinks(root) {
  if (!root) {
    return;
  }
  const anchors = root.querySelectorAll("a[href]");
  if (!anchors.length) {
    return;
  }

  anchors.forEach((anchor) => {
    if (anchor.dataset.normalisedLink === "true") {
      return;
    }

    const href = anchor.getAttribute("href");
    const parsed = parseInternalDocumentLink(href);
    if (!parsed) {
      return;
    }

    anchor.dataset.normalisedLink = "true";
    anchor.dataset.pagePath = parsed.path;

    const encodedPath = encodeURIComponent(parsed.path);
    const pageRoute = `/page/${parsed.path}`;
    const pushValue = parsed.hash ? `${pageRoute}${parsed.hash}` : pageRoute;

    if (window.htmx) {
      anchor.setAttribute("href", pushValue);
      anchor.setAttribute("hx-get", `/api/page/${encodedPath}`);
      anchor.setAttribute("hx-target", "#page-region");
      anchor.setAttribute("hx-swap", "innerHTML");
      anchor.setAttribute("hx-push-url", pushValue);
    } else {
      anchor.removeAttribute("hx-get");
      anchor.removeAttribute("hx-target");
      anchor.removeAttribute("hx-swap");
      anchor.removeAttribute("hx-push-url");
      const htmlTarget = toHtmlRelative(parsed.path);
      anchor.setAttribute("href", parsed.hash ? `${htmlTarget}${parsed.hash}` : htmlTarget);
    }
  });
}

function parseInternalDocumentLink(href) {
  if (!href || typeof href !== "string") {
    return null;
  }
  const trimmed = href.trim();
  if (!trimmed || trimmed.startsWith("#")) {
    return null;
  }

  const hasProtocol = /^[a-zA-Z][a-zA-Z\d+.-]*:/.test(trimmed);
  let url;
  try {
    url = new URL(trimmed, window.location.href);
  } catch {
    return null;
  }

  if (hasProtocol && url.origin !== window.location.origin) {
    return null;
  }

  // Check for both /page/ and /api/page/ prefixes
  const pagePrefix = "/page/";
  const apiPrefix = "/api/page/";
  let docPath = null;

  if (url.pathname.startsWith(pagePrefix)) {
    docPath = url.pathname.slice(pagePrefix.length);
  } else if (url.pathname.startsWith(apiPrefix)) {
    docPath = url.pathname.slice(apiPrefix.length);
  } else {
    return null;
  }

  if (!docPath) {
    return null;
  }

  try {
    docPath = decodeURIComponent(docPath);
  } catch {
    docPath = docPath.replace(/%2F/gi, "/");
  }

  docPath = docPath.replace(/^\/+/, "");
  if (!docPath) {
    return null;
  }

  return {
    path: docPath,
    hash: url.hash || "",
  };
}

function toHtmlRelative(path) {
  let clean = (path || "").trim();
  if (!clean) {
    return "index.html";
  }
  clean = clean.replace(/\.(md|markdown)$/i, "");
  clean = clean.replace(/\/+$/, "");
  if (!clean) {
    return "index.html";
  }
  return `${clean}.html`;
}

function addCopyButtonsToCodeBlocks(root) {
  if (!root) {
    return;
  }

  const codeBlocks = root.querySelectorAll('pre:not(.code-block-wrapper)');

  codeBlocks.forEach((pre) => {
    // Wrap pre in a wrapper div
    const wrapper = document.createElement('div');
    wrapper.className = 'code-block-wrapper';
    pre.parentNode.insertBefore(wrapper, pre);
    wrapper.appendChild(pre);

    // Create copy button
    const button = document.createElement('button');
    button.className = 'code-copy-button';
    button.innerHTML = `
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
        <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
      </svg>
      <span>Copy</span>
    `;
    button.setAttribute('aria-label', 'Copy code to clipboard');

    // Add click handler
    button.addEventListener('click', async () => {
      const code = pre.querySelector('code');
      const text = code ? code.textContent : pre.textContent;

      try {
        await navigator.clipboard.writeText(text);
        button.classList.add('copied');
        button.innerHTML = `
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="20 6 9 17 4 12"></polyline>
          </svg>
          <span>Copied!</span>
        `;

        setTimeout(() => {
          button.classList.remove('copied');
          button.innerHTML = `
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            </svg>
            <span>Copy</span>
          `;
        }, 2000);
      } catch (err) {
        console.error('Failed to copy:', err);
      }
    });

    wrapper.appendChild(button);
  });
}
