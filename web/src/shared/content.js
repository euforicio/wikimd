export function enhanceContent(element) {
  if (!element) {
    return;
  }

  normaliseInternalLinks(element);
  addCopyButtonsToCodeBlocks(element);
  renderMermaid(element);
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

    // Dark mode theme variables - comprehensive Gantt support
    const darkThemeVars = {
      // Core colors
      primaryColor: "#1e293b",
      primaryTextColor: "#f1f5f9",
      primaryBorderColor: "#475569",

      // Background
      background: "#0f172a",
      mainBkg: "#1e293b",
      secondBkg: "#334155",

      // Text - comprehensive coverage
      titleColor: "#f1f5f9",
      textColor: "#f1f5f9",
      labelTextColor: "#f1f5f9",
      darkTextColor: "#f1f5f9",
      loopTextColor: "#f1f5f9",

      // Gantt section backgrounds
      sectionBkgColor: "#1e293b",
      sectionBkgColor2: "#334155",
      altSectionBkgColor: "#475569",

      // Grid and lines
      gridColor: "#475569",
      lineColor: "#64748b",
      todayLineColor: "#ef4444",

      // Gantt text - CRITICAL for visibility
      ganttSectionTextColor: "#f1f5f9",
      taskTextColor: "#f1f5f9",
      taskTextOutsideColor: "#f1f5f9",
      taskTextLightColor: "#0f172a",
      taskTextClickableColor: "#60a5fa",

      // Axis labels
      axisTextColor: "#f1f5f9",

      // Tasks - different states
      activeTaskBkgColor: "#3b82f6",
      activeTaskBorderColor: "#2563eb",
      doneTaskBkgColor: "#10b981",
      doneTaskBorderColor: "#059669",
      critBkgColor: "#ef4444",
      critBorderColor: "#dc2626",
      taskBkgColor: "#475569",
      taskBorderColor: "#64748b",
      excludeBkgColor: "#7f1d1d",

      // Legend
      legendTextColor: "#f1f5f9",

      // Sequence diagram elements
      actorBkg: "#475569",
      actorBorder: "#64748b",
      actorTextColor: "#f1f5f9",
      actorLineColor: "#64748b",
      signalColor: "#f1f5f9",
      signalTextColor: "#f1f5f9",
      labelBoxBkgColor: "#475569",
      labelBoxBorderColor: "#64748b",
      labelTextColor: "#f1f5f9",
      loopTextColor: "#f1f5f9",
      noteBkgColor: "#475569",
      noteTextColor: "#f1f5f9",
      noteBorderColor: "#64748b",
      activationBkgColor: "#475569",
      activationBorderColor: "#64748b",
      sequenceNumberColor: "#0f172a",
      errorBkgColor: "#991b1b",
      errorTextColor: "#fecaca",
      classText: "#f1f5f9",
      mainContrastColor: "#f1f5f9",
      secondaryColor: "#475569",
      tertiaryColor: "#334155",

      // Typography
      fontFamily: "ui-sans-serif, system-ui, sans-serif",
      fontSize: "14px",

      // Fill colors for different diagram types
      fillType0: "#60a5fa",
      fillType1: "#a78bfa",
      fillType2: "#34d399",
      fillType3: "#fbbf24",
      fillType4: "#f87171",
      fillType5: "#f472b6",
      fillType6: "#2dd4bf",
      fillType7: "#fb923c",
    };

    // Detect current theme
    const isDark = document.documentElement.classList.contains("dark");

    // Initialize mermaid with appropriate theme
    window.mermaid.initialize({
      startOnLoad: false,
      theme: isDark ? "dark" : "default",
      themeVariables: isDark ? darkThemeVars : {},
    });

    // Process each mermaid element
    for (const el of mermaidElements) {
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
