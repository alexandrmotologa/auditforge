const path = require('path');
const fs = require('fs');
const { execSync } = require('child_process');
const puppeteer = require('B:/workgit/node_modules/puppeteer');

const OUT_DIR = path.resolve(__dirname, '../docs/images');
const FRAMES_DIR = path.resolve(__dirname, '../docs/images/frames');

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function capture() {
  if (!fs.existsSync(OUT_DIR)) {
    fs.mkdirSync(OUT_DIR, { recursive: true });
  }
  if (!fs.existsSync(FRAMES_DIR)) {
    fs.mkdirSync(FRAMES_DIR, { recursive: true });
  }

  console.log('Launching Puppeteer headless browser...');
  const browser = await puppeteer.launch({
    headless: true,
    defaultViewport: null,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--force-device-scale-factor=1']
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900, deviceScaleFactor: 2 });

  console.log('Navigating to http://localhost:8080/ui/ ...');
  await page.goto('http://localhost:8080/ui/', { waitUntil: 'domcontentloaded', timeout: 15000 });
  await sleep(2000);

  // 1. Capture Main Dashboard
  console.log('1. Capturing auditforge_dashboard.png (1440x900 @2x)...');
  await page.screenshot({
    path: path.join(OUT_DIR, 'auditforge_dashboard.png'),
    fullPage: false,
  });

  // 2. Click first event to open EventDetailModal (Merkle Proof & State Mutation Diff)
  console.log('2. Opening Event Detail Modal...');
  const firstEventRow = await page.$('tbody tr:first-child');
  if (firstEventRow) {
    await firstEventRow.click();
    await sleep(1000);

    console.log('Capturing auditforge_event_detail.png...');
    await page.screenshot({
      path: path.join(OUT_DIR, 'auditforge_event_detail.png'),
      fullPage: false,
    });

    // Close modal by clicking X or Close button
    const closeBtn = await page.$('button[title="Close modal"], button:has(svg.lucide-x)');
    if (closeBtn) {
      await closeBtn.click();
    } else {
      await page.keyboard.press('Escape');
    }
    await sleep(800);
  }

  // 3. Switch to Block Chain Explorer
  console.log('3. Opening Block Chain & Merkle DAG Explorer...');
  const blockchainTabBtn = await page.evaluateHandle(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    return btns.find(b => b.innerText.includes('Block Chain & Merkle DAG Explorer'));
  });

  if (blockchainTabBtn && blockchainTabBtn.asElement()) {
    await blockchainTabBtn.asElement().click();
    await sleep(1200);

    console.log('Capturing auditforge_blockchain_explorer.png...');
    await page.screenshot({
      path: path.join(OUT_DIR, 'auditforge_blockchain_explorer.png'),
      fullPage: false,
    });
  }

  // Switch back to Events
  const eventsTabBtn = await page.evaluateHandle(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    return btns.find(b => b.innerText.includes('Real-Time Audit Event Stream'));
  });
  if (eventsTabBtn && eventsTabBtn.asElement()) {
    await eventsTabBtn.asElement().click();
    await sleep(800);
  }

  // 4. Capture Tamper Alert Simulation
  console.log('4. Demonstrating Tamper Detection Alert...');
  const tamperBtn = await page.evaluateHandle(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    return btns.find(b => b.innerText.includes('Tamper Demo'));
  });

  if (tamperBtn && tamperBtn.asElement()) {
    await tamperBtn.asElement().click();
    await sleep(800);

    // Click Inject Simulated Tamper
    const injectTamperBtn = await page.evaluateHandle(() => {
      const btns = Array.from(document.querySelectorAll('button'));
      return btns.find(b => b.innerText.includes('Inject Simulated Tamper'));
    });
    if (injectTamperBtn && injectTamperBtn.asElement()) {
      await injectTamperBtn.asElement().click();
      await sleep(1200);
    }

    // Close the lab modal to showcase the red alert banner on the dashboard
    const closeLabBtn = await page.evaluateHandle(() => {
      const btns = Array.from(document.querySelectorAll('button'));
      return btns.find(b => b.innerText.includes('Close Lab'));
    });
    if (closeLabBtn && closeLabBtn.asElement()) {
      await closeLabBtn.asElement().click();
      await sleep(1000);
    }

    console.log('Capturing auditforge_tamper_alert.png (Violated Hash Link Alert)...');
    await page.screenshot({
      path: path.join(OUT_DIR, 'auditforge_tamper_alert.png'),
      fullPage: false,
    });

    // Reopen and restore integrity so the system is clean
    const reopenTamper = await page.evaluateHandle(() => {
      const btns = Array.from(document.querySelectorAll('button'));
      return btns.find(b => b.innerText.includes('Tamper Demo'));
    });
    if (reopenTamper && reopenTamper.asElement()) {
      await reopenTamper.asElement().click();
      await sleep(600);

      const restoreBtn = await page.evaluateHandle(() => {
        const btns = Array.from(document.querySelectorAll('button'));
        return btns.find(b => b.innerText.includes('Restore Ledger Integrity'));
      });
      if (restoreBtn && restoreBtn.asElement()) {
        await restoreBtn.asElement().click();
        await sleep(1000);
      }

      const closeAgain = await page.evaluateHandle(() => {
        const btns = Array.from(document.querySelectorAll('button'));
        return btns.find(b => b.innerText.includes('Close Lab'));
      });
      if (closeAgain && closeAgain.asElement()) {
        await closeAgain.asElement().click();
        await sleep(800);
      }
    }
  }

  // 5. Mobile Responsive Screenshot (iPhone 14/15 size: 390x844 @2x)
  console.log('5. Capturing auditforge_mobile.png (390x844 @2x)...');
  await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 2, isMobile: true, hasTouch: true });
  await page.reload({ waitUntil: 'domcontentloaded' });
  await sleep(1500);
  await page.screenshot({
    path: path.join(OUT_DIR, 'auditforge_mobile.png'),
    fullPage: false,
  });

  // 6. Record Animated Frames for auditforge_demo.gif
  console.log('6. Recording interactive flow for auditforge_demo.gif...');
  await page.setViewport({ width: 1280, height: 760, deviceScaleFactor: 1 });
  await page.reload({ waitUntil: 'domcontentloaded' });
  await sleep(1200);

  // Clean old frames
  const oldFrames = fs.readdirSync(FRAMES_DIR);
  for (const f of oldFrames) {
    fs.unlinkSync(path.join(FRAMES_DIR, f));
  }

  let frameIdx = 0;
  async function grab(count = 1) {
    for (let i = 0; i < count; i++) {
      const pad = String(frameIdx++).padStart(3, '0');
      await page.screenshot({ path: path.join(FRAMES_DIR, `frame_${pad}.png`) });
      await sleep(100);
    }
  }

  // Frame sequence:
  console.log('Recording frames...');
  // A. Initial Dashboard hold (10 frames)
  await grab(10);

  // B. Click "🔐 Auth & MFA" quick filter
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const btn = btns.find(b => b.innerText.includes('Auth & MFA'));
    if (btn) btn.click();
  });
  await grab(8);

  // C. Click "🛡️ Access & Roles" quick filter
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const btn = btns.find(b => b.innerText.includes('Access & Roles'));
    if (btn) btn.click();
  });
  await grab(8);

  // D. Click first event to open Merkle Proof inspector
  const row = await page.$('tbody tr:first-child');
  if (row) {
    await row.click();
    await grab(12);

    // Scroll modal down slightly to reveal Merkle proof tree and State diff
    await page.evaluate(() => {
      const modalBody = document.querySelector('.overflow-y-auto');
      if (modalBody) modalBody.scrollBy({ top: 240, behavior: 'smooth' });
    });
    await grab(12);

    // Close modal
    await page.keyboard.press('Escape');
    await grab(6);
  }

  // E. Switch to Block Chain Explorer
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const tab = btns.find(b => b.innerText.includes('Block Chain & Merkle DAG Explorer'));
    if (tab) tab.click();
  });
  await grab(12);

  // Scroll down slightly in Block Chain Explorer
  await page.evaluate(() => window.scrollBy({ top: 300, behavior: 'smooth' }));
  await grab(10);

  // Scroll back up and switch back to events
  await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'smooth' }));
  await grab(6);

  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const tab = btns.find(b => b.innerText.includes('Real-Time Audit Event Stream'));
    if (tab) tab.click();
  });
  await grab(8);

  await browser.close();

  // 7. Compile frames into animated GIF using ffmpeg
  console.log(`7. Compiling ${frameIdx} frames with ffmpeg into auditforge_demo.gif...`);
  const gifPath = path.join(OUT_DIR, 'auditforge_demo.gif');
  const ffmpegCmd = `ffmpeg -y -framerate 10 -i "${path.join(FRAMES_DIR, 'frame_%03d.png')}" -vf "fps=10,scale=1280:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128:reserve_transparent=0[p];[s1][p]paletteuse=dither=bayer:bayer_scale=5" "${gifPath}"`;

  execSync(ffmpegCmd, { stdio: 'inherit' });
  console.log('Generated GIF: ' + gifPath + ' (' + fs.statSync(gifPath).size + ' bytes)');

  // Clean frames directory
  for (const f of fs.readdirSync(FRAMES_DIR)) {
    fs.unlinkSync(path.join(FRAMES_DIR, f));
  }
  fs.rmdirSync(FRAMES_DIR);

  console.log('All authentic visual assets captured successfully!');
}

capture().catch(err => {
  console.error('Asset capture failed:', err);
  process.exit(1);
});
