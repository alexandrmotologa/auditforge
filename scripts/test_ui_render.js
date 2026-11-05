const puppeteer = require('B:/workgit/node_modules/puppeteer');

(async () => {
  const browser = await puppeteer.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });
  const page = await browser.newPage();
  page.on('console', msg => console.log('BROWSER LOG:', msg.text()));
  page.on('pageerror', err => console.error('BROWSER ERROR:', err.message));
  page.on('requestfailed', req => console.log('REQ FAILED:', req.url(), req.failure()?.errorText));
  page.on('response', res => {
    if (res.status() >= 400) {
      console.log('HTTP ERROR:', res.status(), res.url());
    }
  });

  await page.setViewport({ width: 1440, height: 900 });
  try {
    await page.goto('http://localhost:8080/ui/', { waitUntil: 'domcontentloaded', timeout: 5000 });
    await new Promise(r => setTimeout(r, 1000));
    const title = await page.title();
    console.log('PAGE TITLE:', title);
    const text = await page.evaluate(() => document.body.innerText);
    console.log('PAGE BODY LENGTH:', text.length);
    console.log('SNIPPET:', text.slice(0, 200));
  } catch (err) {
    console.error('NAV ERROR:', err);
  }

  await browser.close();
})();
