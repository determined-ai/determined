import fs from 'fs/promises';
import path from 'path';

import { chromium } from 'playwright-core';
import { createServer } from 'vite';

const THEMES = ['light', 'dark'];

const label = process.argv[2] || '';
const screenPath = path.resolve(process.cwd(), ...['screenshots', label].filter((c) => c));

// start dev server
const devServer = await createServer({
  mode: 'test', // disables typescript checking
  server: {
    open: false,
    port: 3456,
  },
});
await devServer.listen();

// handle assembling the base url
const publicUrl = process.env['PUBLIC_URL'] || '';
const { address, port } = devServer.httpServer.address();

// start chrome playwright
const browser = await chromium.launch();
const page = await browser.newPage();
await page.goto(`http://${address}:${port}${publicUrl}/design/?exclusive=true`);
// take screenshots of each section
const links = await page.locator('nav ul a').all();
if (links.length === 0) {
  console.error('WARNING: No sections found');
}

for (const theme of THEMES) {
  const themePath = path.resolve(screenPath, theme);
  await fs.mkdir(themePath, { recursive: true });
  await page.emulateMedia({ colorScheme: theme });
  for (const link of links) {
    await link.click();
    const title = await link.innerText();
    const section = page.locator('nav + article');
    // playwright hangs if height is a non-int
    const height = Math.ceil((await section.boundingBox()).height);
    await page.setViewportSize({ height, width: 1280 });
    await section.screenshot({
      animations: 'disabled',
      path: path.resolve(themePath, `${title}.png`),
    });
  }
}

// clean up
await browser.close();
await devServer.close();
