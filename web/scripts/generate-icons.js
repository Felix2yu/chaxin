import sharp from 'sharp';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const __dirname = dirname(fileURLToPath(import.meta.url));
const srcIcon = join(__dirname, '..', 'public', 'icon.svg');
const outDir = join(__dirname, '..', 'public');

const sizes = [192, 512];

async function generateIcons() {
  for (const size of sizes) {
    const outPath = join(outDir, `icon-${size}x${size}.png`);
    await sharp(srcIcon)
      .resize(size, size)
      .png()
      .toFile(outPath);
    console.log(`Generated ${outPath}`);
  }
}

generateIcons().catch(console.error);
