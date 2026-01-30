// Playwright launchServer for Windows
const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

const args = process.argv.slice(2);
const host = args.find(a => a.startsWith('--host='))?.split('=')[1] || '0.0.0.0';
const port = parseInt(args.find(a => a.startsWith('--port='))?.split('=')[1] || '9323');

const wsPathFile = path.join(__dirname, 'ws_path.txt');
const wsPath = fs.existsSync(wsPathFile) ? fs.readFileSync(wsPathFile, 'utf8').trim() : undefined;

(async () => {
  const server = await chromium.launchServer({
    host,
    port,
    wsPath,
  });
  console.log(`Playwright server started: ${server.wsEndpoint()}`);
})();
