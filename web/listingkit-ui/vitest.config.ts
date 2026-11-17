import { configDefaults, defineConfig } from "vitest/config";
import path from "node:path";

const nodeTests = ["src/lib/{api,contracts,server}/**/*.test.ts"];

export default defineConfig({
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    css: true,
    maxWorkers: 4,
    testTimeout: 15000,
    server: { deps: { inline: ["next-auth"] } },
    exclude: [
      ...configDefaults.exclude,
      "e2e/**",
      ".next/**",
      "coverage/**",
      "test-results/**",
      ".playwright-cli/**",
      ".data/**",
    ],
    projects: [
      {
        test: {
          // Files that exercise browser APIs can opt into jsdom with @vitest-environment.
          name: "libraries",
          environment: "node",
          include: nodeTests,
        },
      },
      {
        test: {
          name: "app",
          exclude: nodeTests,
        },
      },
    ],
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "next/server": path.resolve(__dirname, "./node_modules/next/server.js"),
      "next/headers": path.resolve(__dirname, "./node_modules/next/headers.js"),
      "next/navigation": path.resolve(__dirname, "./node_modules/next/navigation.js"),
    },
  },
});
