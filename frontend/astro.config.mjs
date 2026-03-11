// @ts-check
import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';

// https://astro.build/config
export default defineConfig({
  output: 'static', // Cloudflare Pages用
  vite: {
    plugins: [tailwindcss()],
  },
});