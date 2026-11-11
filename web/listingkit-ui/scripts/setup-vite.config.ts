import { defineConfig } from "vite";
import path from "node:path";

export default defineConfig({
  root: path.resolve(import.meta.dirname, "../src/setup"),
  resolve: { alias: { "@": path.resolve(import.meta.dirname, "../src") } },
  css: { postcss: path.resolve(import.meta.dirname, "..") },
  build: { outDir: path.resolve(import.meta.dirname, "../.setup-dist"), emptyOutDir: true },
});
