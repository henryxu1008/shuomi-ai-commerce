import {defineConfig,devices} from "@playwright/test";

const origin=process.env.OWN_INTAKE_BROWSER_ORIGIN;
if(!origin||!/^http:\/\/127\.0\.0\.1:\d+$/.test(origin))throw new Error("A dedicated loopback own-intake origin is required");

export default defineConfig({
 testDir:"./e2e",
 testMatch:"own-product-intake.spec.ts",
 outputDir:"../../.local/playwright/own-intake-results",
 reporter:"line",
 use:{baseURL:origin,screenshot:"only-on-failure",trace:"retain-on-failure"},
 projects:[{name:"chromium",use:{...devices["Desktop Chrome"]}}],
 webServer:{command:`pnpm dev --hostname 127.0.0.1 --port ${new URL(origin).port}`,url:origin,reuseExistingServer:false,timeout:120000},
});
