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
        },
        {
          src: 'styles/main.css',
          dest: '.'
        },
        {
          src: 'styles/dify.css',
          dest: '.'
        }
      ]
    })
  ],
  root: 'src',
  publicDir: '../public',
  build: {
    outDir: '../tmp',
    emptyOutDir: true,
    rollupOptions: {
      input: 'src/build-entry.js',
      output: {
        entryFileNames: 'build-dummy.js'
      }
    }
  },
  server: {
    port: 3000,
    open: true,
  },
})
