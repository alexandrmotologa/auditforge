const fs = require('fs');
const path = require('path');
const { Resvg } = require(path.resolve(__dirname, '../web/node_modules/@resvg/resvg-js'));

function generateLogoSVG() {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="1024" height="1024">
  <defs>
    <!-- Background Gradient for Hexagon Vault -->
    <linearGradient id="vaultBg" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#0b1120" />
      <stop offset="50%" stop-color="#0f172a" />
      <stop offset="100%" stop-color="#020617" />
    </linearGradient>

    <!-- Emerald / Cyan Eye Core Gradient -->
    <linearGradient id="eyeGlow" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#00f5ff" />
      <stop offset="50%" stop-color="#10b981" />
      <stop offset="100%" stop-color="#059669" />
    </linearGradient>

    <!-- Emerald Armor Accent Gradient -->
    <linearGradient id="armorAccent" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#10b981" />
      <stop offset="100%" stop-color="#0284c7" />
    </linearGradient>

    <!-- Subtle Radial Gradient for Vault Depth -->
    <radialGradient id="vaultGlow" cx="50%" cy="45%" r="65%">
      <stop offset="0%" stop-color="#10b981" stop-opacity="0.12" />
      <stop offset="70%" stop-color="#00f5ff" stop-opacity="0.02" />
      <stop offset="100%" stop-color="#000000" stop-opacity="0" />
    </radialGradient>
  </defs>

  <!-- 1. Luxury White Squircle Container -->
  <rect x="16" y="16" width="992" height="992" rx="220" fill="#ffffff" stroke="#e2e8f0" stroke-width="8" />

  <!-- Subtle Outer Container Grid Inset -->
  <rect x="40" y="40" width="944" height="944" rx="196" fill="none" stroke="#f1f5f9" stroke-width="2" />

  <!-- 2. Massive Hexagonal Vault Frame (Center at 512, 512) -->
  <!-- Outer Solid Hexagon Base -->
  <polygon points="512,140 850,335 850,725 512,920 174,725 174,335"
           fill="url(#vaultBg)" stroke="#0f172a" stroke-width="36" stroke-linejoin="round" />

  <!-- Inner Vault Radial Glow -->
  <polygon points="512,140 850,335 850,725 512,920 174,725 174,335"
           fill="url(#vaultGlow)" />

  <!-- Inner Cyan / Emerald Cryptographic Perimeter Grid -->
  <polygon points="512,168 824,348 824,708 512,888 200,708 200,348"
           fill="none" stroke="#10b981" stroke-width="3" stroke-dasharray="16 12" stroke-opacity="0.45" />

  <!-- 3. Watertight Solid Obsidian Base Silhouette (Prevents any background leaks) -->
  <path d="M 512,210
           L 570,250 L 610,310 L 670,360 L 730,440 L 750,540 L 730,650 L 680,740 L 590,820 L 512,850
           L 434,820 L 344,740 L 294,650 L 274,540 L 294,440 L 354,360 L 414,310 L 454,250 Z"
        fill="#070b14" />

  <!-- 4. Low-Poly Volumetric Facets of the Armored Sentinel Mascot -->

  <!-- CROWN & CREST (Top Head Plates) -->
  <polygon points="512,210 460,260 512,300" fill="#334155" />
  <polygon points="512,210 512,300 564,260" fill="#475569" />

  <polygon points="460,260 410,320 470,340 512,300" fill="#1e293b" />
  <polygon points="564,260 512,300 554,340 614,320" fill="#334155" />

  <!-- TEMPLE & EAR GUARDS -->
  <polygon points="410,320 340,360 410,390 470,340" fill="#0f172a" />
  <polygon points="614,320 554,340 614,390 684,360" fill="#1e293b" />

  <polygon points="340,360 290,430 360,450 410,390" fill="#1e293b" />
  <polygon points="684,360 614,390 664,450 734,430" fill="#334155" />

  <!-- FOREHEAD MERKLE ROOT PRISM -->
  <polygon points="512,300 470,340 512,380" fill="#1e293b" />
  <polygon points="512,300 512,380 554,340" fill="#334155" />

  <polygon points="470,340 440,390 512,380" fill="#0f172a" />
  <polygon points="554,340 512,380 584,390" fill="#1e293b" />

  <!-- CENTRAL BROW RIDGE -->
  <polygon points="512,380 470,430 512,450" fill="#334155" />
  <polygon points="512,380 512,450 554,430" fill="#475569" />

  <!-- PREDATORY EMERALD/CYAN EYES (Sharp Polygon Sockets) -->
  <!-- Left Eye Socket & Pupil -->
  <polygon points="430,395 475,410 440,425" fill="#020617" />
  <polygon points="436,400 470,412 444,422" fill="url(#eyeGlow)" />

  <!-- Right Eye Socket & Pupil -->
  <polygon points="594,395 584,425 549,410" fill="#020617" />
  <polygon points="588,400 580,422 554,412" fill="url(#eyeGlow)" />

  <!-- EYE CANTHUS ACCENTS -->
  <polygon points="425,393 430,395 440,425 420,415" fill="#00f5ff" opacity="0.6" />
  <polygon points="599,393 604,415 584,425 594,395" fill="#00f5ff" opacity="0.6" />

  <!-- SNOUT & MUZZLE (Titanium Slate, Horizontal Distribution) -->
  <polygon points="512,450 470,430 460,490 512,520" fill="#1e293b" />
  <polygon points="512,450 512,520 564,490 554,430" fill="#334155" />

  <polygon points="460,490 480,540 512,520" fill="#334155" />
  <polygon points="564,490 512,520 544,540" fill="#475569" />

  <!-- NOSE APEX KEYSTONE -->
  <polygon points="512,520 495,545 512,555 529,545" fill="#0f172a" />
  <polygon points="512,510 500,535 512,545 524,535" fill="#10b981" />

  <!-- CHEEK FLANGES (Volumetric Lateral Armor) -->
  <polygon points="410,390 360,450 420,480 460,430" fill="#1e293b" />
  <polygon points="614,390 564,430 604,480 664,450" fill="#334155" />

  <polygon points="360,450 310,530 380,540 420,480" fill="#0f172a" />
  <polygon points="664,450 604,480 644,540 714,530" fill="#1e293b" />

  <polygon points="420,480 380,540 440,570 480,540 460,490" fill="#334155" />
  <polygon points="604,480 564,490 544,540 584,570 644,540" fill="#475569" />

  <!-- JAW SHIELD & GULAR PLATES -->
  <polygon points="512,555 460,575 512,610" fill="#0f172a" />
  <polygon points="512,555 512,610 564,575" fill="#1e293b" />

  <polygon points="460,575 420,620 480,635 512,610" fill="#1e293b" />
  <polygon points="564,575 512,610 544,635 604,620" fill="#334155" />

  <!-- INTERLOCKING MERKLE DAG CHEST ARMOR (Layer 1) -->
  <polygon points="512,610 440,650 512,690" fill="#10b981" opacity="0.9" />
  <polygon points="512,610 512,690 584,650" fill="#059669" opacity="0.9" />

  <polygon points="440,650 370,680 430,720 512,690" fill="#1e293b" />
  <polygon points="584,650 512,690 594,720 654,680" fill="#334155" />

  <!-- SHOULDER PAULDRONS & LATERAL SCALE ROWS (Layer 2) -->
  <polygon points="380,540 310,530 300,630 370,640" fill="#1e293b" />
  <polygon points="644,540 654,640 724,630 714,530" fill="#334155" />

  <polygon points="370,640 300,630 320,720 390,710" fill="#0f172a" />
  <polygon points="654,640 634,710 704,720 724,630" fill="#1e293b" />

  <polygon points="390,710 320,720 360,790 420,770" fill="#1e293b" />
  <polygon points="634,710 604,770 664,790 704,720" fill="#334155" />

  <!-- INFERIOR BASAL PLATES & TAIL APEX (Layer 3 - Ground Anchor) -->
  <polygon points="512,690 430,720 480,780 512,760" fill="#334155" />
  <polygon points="512,690 512,760 544,780 594,720" fill="#475569" />

  <polygon points="512,760 450,800 512,840" fill="#1e293b" />
  <polygon points="512,760 512,840 574,800" fill="#334155" />

  <polygon points="480,780 420,810 470,845 512,840" fill="#0f172a" />
  <polygon points="544,780 512,840 554,845 604,810" fill="#1e293b" />

  <!-- CRYPTOGRAPHIC ACCENT EDGES & MERKLE ROOTS -->
  <line x1="512" y1="210" x2="512" y2="300" stroke="#00f5ff" stroke-width="2.5" stroke-opacity="0.8" />
  <line x1="512" y1="300" x2="512" y2="380" stroke="#10b981" stroke-width="2.5" stroke-opacity="0.8" />
  <line x1="512" y1="610" x2="512" y2="690" stroke="#00f5ff" stroke-width="3" stroke-opacity="0.9" />
  <line x1="512" y1="760" x2="512" y2="840" stroke="#10b981" stroke-width="2" stroke-opacity="0.7" />

  <!-- Subtle Vertex Anchors -->
  <circle cx="512" cy="210" r="3.5" fill="#00f5ff" />
  <circle cx="512" cy="380" r="3" fill="#10b981" />
  <circle cx="512" cy="610" r="4" fill="#00f5ff" />
  <circle cx="512" cy="690" r="4" fill="#10b981" />
  <circle cx="512" cy="840" r="3" fill="#00f5ff" />
</svg>`;
}

async function render() {
  const svg = generateLogoSVG();
  const outDir = path.resolve(__dirname, '../docs/images');
  if (!fs.existsSync(outDir)) {
    fs.mkdirSync(outDir, { recursive: true });
  }

  // Write SVG
  const svgPath = path.join(outDir, 'logo.svg');
  fs.writeFileSync(svgPath, svg, 'utf8');
  console.log('Saved SVG: ' + svgPath);

  // Render PNG with Resvg at 1024x1024
  const resvg = new Resvg(svg, {
    fitTo: { mode: 'width', value: 1024 },
  });
  const pngData = resvg.render();
  const pngBuffer = pngData.asPng();

  const pngPath = path.join(outDir, 'logo.png');
  fs.writeFileSync(pngPath, pngBuffer);
  console.log('Saved PNG: ' + pngPath + ' (' + pngBuffer.length + ' bytes)');
}

render().catch(console.error);
