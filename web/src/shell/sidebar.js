// Sidebar resize and collapse functionality

const MIN_SIDEBAR_WIDTH = 200;
const MAX_SIDEBAR_WIDTH = 600;
const DEFAULT_SIDEBAR_WIDTH = 320;
const STORAGE_KEY_WIDTH = 'wikimd_sidebar_width';
const STORAGE_KEY_COLLAPSED = 'wikimd_sidebar_collapsed';

export function initSidebar() {
  const sidebar = document.getElementById('sidebar');
  const resizeHandle = document.getElementById('sidebar-resize-handle');
  const collapseBtn = document.getElementById('sidebar-collapse-btn');

  if (!sidebar || !resizeHandle || !collapseBtn) {
    return;
  }

  // Load saved state
  const savedWidth = localStorage.getItem(STORAGE_KEY_WIDTH);
  const savedCollapsed = localStorage.getItem(STORAGE_KEY_COLLAPSED) === 'true';

  // Apply saved width
  if (savedWidth) {
    sidebar.style.width = savedWidth + 'px';
  }

  // Apply saved collapsed state
  if (savedCollapsed) {
    collapseSidebar(sidebar, collapseBtn, false);
  }

  // Resize functionality with smooth performance
  let isResizing = false;
  let startX = 0;
  let startWidth = 0;
  let animationFrameId = null;

  resizeHandle.addEventListener('mousedown', (e) => {
    isResizing = true;
    startX = e.clientX;
    startWidth = sidebar.offsetWidth;
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    e.preventDefault();
  });

  document.addEventListener('mousemove', (e) => {
    if (!isResizing) return;

    // Use requestAnimationFrame for smooth resizing
    if (animationFrameId) {
      cancelAnimationFrame(animationFrameId);
    }

    animationFrameId = requestAnimationFrame(() => {
      const delta = e.clientX - startX;
      const newWidth = Math.max(MIN_SIDEBAR_WIDTH, Math.min(MAX_SIDEBAR_WIDTH, startWidth + delta));
      sidebar.style.width = newWidth + 'px';
    });
  });

  document.addEventListener('mouseup', () => {
    if (isResizing) {
      isResizing = false;
      document.body.style.cursor = '';
      document.body.style.userSelect = '';

      if (animationFrameId) {
        cancelAnimationFrame(animationFrameId);
        animationFrameId = null;
      }

      // Save width to localStorage
      localStorage.setItem(STORAGE_KEY_WIDTH, sidebar.offsetWidth);
    }
  });

  // Collapse functionality
  collapseBtn.addEventListener('click', () => {
    const isCollapsed = sidebar.classList.contains('collapsed');

    if (isCollapsed) {
      expandSidebar(sidebar, collapseBtn);
    } else {
      collapseSidebar(sidebar, collapseBtn);
    }
  });
}

function collapseSidebar(sidebar, btn, animate = true) {
  if (!animate) {
    sidebar.classList.add('no-transition');
  }

  sidebar.classList.add('collapsed');
  sidebar.style.width = '0px';
  sidebar.style.minWidth = '0px';
  sidebar.style.overflow = 'hidden';

  const icon = btn.querySelector('svg');
  if (icon) {
    icon.style.transform = 'rotate(180deg)';
  }

  localStorage.setItem(STORAGE_KEY_COLLAPSED, 'true');

  if (!animate) {
    // Force reflow then remove no-transition
    sidebar.offsetHeight;
    sidebar.classList.remove('no-transition');
  }
}

function expandSidebar(sidebar, btn) {
  sidebar.classList.remove('collapsed');

  // Restore saved width or use default
  const savedWidth = localStorage.getItem(STORAGE_KEY_WIDTH);
  const width = savedWidth || DEFAULT_SIDEBAR_WIDTH;
  sidebar.style.width = width + 'px';
  sidebar.style.minWidth = '';
  sidebar.style.overflow = '';

  const icon = btn.querySelector('svg');
  if (icon) {
    icon.style.transform = 'rotate(0deg)';
  }

  localStorage.setItem(STORAGE_KEY_COLLAPSED, 'false');
}
