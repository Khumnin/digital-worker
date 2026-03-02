import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  retries: 0,
  workers: 1, // serial — tests share a live DB tenant

  use: {
    baseURL: process.env.BASE_URL ?? "http://localhost:8080",
    extraHTTPHeaders: {
      "Content-Type": "application/json",
    },
  },

  projects: [
    {
      name: "api",
      // API-only mode — no browser launched
      testMatch: /(?<!\.ui)\.spec\.ts$/,
    },
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
      testMatch: /\.ui\.spec\.ts$/,
    },
  ],

  reporter: [["list"], ["html", { outputFolder: "playwright-report", open: "never" }]],
});
