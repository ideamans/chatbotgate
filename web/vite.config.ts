import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import { viteStaticCopy } from 'vite-plugin-static-copy'

export default defineConfig({
  plugins: [
    tailwindcss(),
    viteStaticCopy({
      targets: [
        {
          src: '../public/icons/*.svg',
          dest: 'icons'
        }
      ]
    })
  ],
  root: 'src',
  publicDir: '../public',
  build: {
    outDir: '../../pkg/middleware/assets/static',
    emptyOutDir: false, // Don't delete existing files (like embedded Go files)
    assetsDir: 'assets',
    rollupOptions: {
      output: {
        // CSS output to specific location for Go embed
        assetFileNames: (assetInfo) => {
          if (assetInfo.name?.endsWith('.css')) {
            return 'styles.css'
          }
          return 'assets/[name]-[hash][extname]'
        },
      },
    },
    // Don't minify for better readability and debugging
    minify: false,
  },
  server: {
    port: 3000,
    open: true,
  },
})
