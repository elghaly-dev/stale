import { chromium } from "playwright";
import { mkdirSync } from "fs";

const outDir = "/opt/cursor/artifacts/screenshots";
mkdirSync(outDir, { recursive: true });

const browser = await chromium.launch({ headless: true });
for (const width of [390, 1280]) {
  const page = await browser.newPage({ viewport: { width, height: 900 } });
  await page.goto("http://127.0.0.1:8765/", { waitUntil: "networkidle" });
  const scrollW = await page.evaluate(() => document.documentElement.scrollWidth);
  const clientW = await page.evaluate(() => document.documentElement.clientWidth);
  console.log(`width ${width}: horizontal overflow`, scrollW > clientW);
  await page.screenshot({ path: `${outDir}/stale-${width}px.png`, fullPage: true });
  await page.close();
}
await browser.close();
console.log("screenshots saved to", outDir);
