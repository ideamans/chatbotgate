import { copyFileSync, mkdirSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));

const pkgDir = join(__dirname, '../pkg/middleware/assets/static');
const tmpDir = join(__dirname, 'tmp');

// Ensure pkg directories exist
mkdirSync(pkgDir, { recursive: true });
mkdirSync(join(pkgDir, 'icons'), { recursive: true });

// Copy CSS files
console.log('Copying CSS files...');
copyFileSync(join(tmpDir, 'main.css'), join(pkgDir, 'main.css'));
copyFileSync(join(tmpDir, 'dify.css'), join(pkgDir, 'dify.css'));

// Copy icons
console.log('Copying icons...');
const icons = [
  'chatbotgate.svg',
  'email.svg',
  'facebook.svg',
  'github.svg',
  'google.svg',
  'microsoft.svg',
  'oidc.svg'
];

icons.forEach(icon => {
  copyFileSync(
    join(tmpDir, 'icons', icon),
    join(pkgDir, 'icons', icon)
  );
});

console.log('✓ Build assets copied to pkg/middleware/assets/static');
