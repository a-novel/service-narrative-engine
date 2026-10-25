import { resolve } from "node:path";

import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    globals: true,
    environment: "jsdom",
    alias: {
      "@a-novel/service-narrative-engine-rest": resolve("./pkg/js/rest/src/index"),
    },
    coverage: {
      enabled: true,
      clean: true,
      provider: "v8",
      reporter: ["text", "json", "html", "lcov"],
      reportsDirectory: "coverage",
      include: ["pkg/js/rest/src/**/*.{ts,tsx}"],
      allowExternal: true,
    },
    projects: [
      {
        root: "pkg/js/test/rest",
        extends: true,
      },
    ],
  },
});
