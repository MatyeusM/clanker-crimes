// @ts-check
import { defineConfig } from 'astro/config'

// https://astro.build/config
//
// The game is hosted on GitHub Pages as a project site, so every built URL
// lives under the /clanker-crimes subpath. Hardcoded absolute paths in markup
// and client-side JS must go through import.meta.env.BASE_URL instead of
// starting with "/".
export default defineConfig({
  site: 'https://matyeusm.github.io/clanker-crimes/',
  base: '/clanker-crimes',
})
