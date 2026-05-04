import { defineConfig } from "vite";

export default defineConfig({
  // GitHub Pages serves from leodido.github.io/structcli/
  // Local dev uses "/" — override via VITE_BASE env var or --base flag.
  base: process.env.VITE_BASE || "/structcli/",
  build: {
    outDir: "dist",
  },
  server: {
    // Allow any host — needed for Gitpod/Codespaces/tunneled URLs
    allowedHosts: true,
    headers: {
      // Required for SharedArrayBuffer if we ever need it.
      // Not needed now but costs nothing to set.
      "Cross-Origin-Opener-Policy": "same-origin",
      "Cross-Origin-Embedder-Policy": "require-corp",
    },
  },
});
