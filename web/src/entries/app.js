import { initAppShell } from "../shell/app-shell";
import { initSidebar } from "../shell/sidebar";
import { reRenderMermaid } from "../shared/content";

// Expose reRenderMermaid globally for theme toggle
window.reRenderMermaid = reRenderMermaid;

initAppShell();
initSidebar();
