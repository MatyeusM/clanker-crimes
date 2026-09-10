import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { escapeHtml, highlightable } from './files.mjs'

export const LIGHT_THEME = 'ayu-light'
export const DARK_THEME = 'ayu-dark'

const CANDIDATE_LANGS = [
  'rust',
  'typescript',
  'tsx',
  'javascript',
  'jsx',
  'json',
  'toml',
  'yaml',
  'html',
  'css',
  'markdown',
  'bash',
  'astro',
  'plaintext',
]

const EXT_TO_LANG = {
  rs: 'rust',
  ts: 'typescript',
  mts: 'typescript',
  cts: 'typescript',
  tsx: 'tsx',
  js: 'javascript',
  mjs: 'javascript',
  cjs: 'javascript',
  jsx: 'jsx',
  json: 'json',
  toml: 'toml',
  yaml: 'yaml',
  yml: 'yaml',
  html: 'html',
  css: 'css',
  md: 'markdown',
  sh: 'bash',
  astro: 'astro',
}

function plainFallback(code) {
  return `<pre class="shiki"><code>${escapeHtml(code)}</code></pre>`
}

// Renders every highlightable file of a package twice (light + dark theme)
// into <caseDir>/{light,dark}/<relpath>.html, plus a files.json manifest.
export async function highlightPackage(pkgDir, caseDir, files) {
  const { bundledLanguages, bundledThemes, createHighlighter } = await import('shiki')
  if (!(LIGHT_THEME in bundledThemes) || !(DARK_THEME in bundledThemes)) {
    throw new Error(`shiki is missing ${LIGHT_THEME} or ${DARK_THEME}`)
  }

  const targets = highlightable(files)
  const highlighter = await createHighlighter({
    themes: [LIGHT_THEME, DARK_THEME],
    langs: CANDIDATE_LANGS.filter(lang => lang in bundledLanguages),
  })

  try {
    const manifest = []
    for (const file of targets) {
      const code = readFileSync(file.abs, 'utf8')
      const lang = EXT_TO_LANG[file.ext] ?? 'plaintext'
      const outputs = {}
      for (const theme of [LIGHT_THEME, DARK_THEME]) {
        try {
          outputs[theme] = highlighter.codeToHtml(code, { lang, theme })
        } catch {
          outputs[theme] = plainFallback(code)
        }
      }
      const lightFile = join(caseDir, 'light', `${file.rel}.html`)
      const darkFile = join(caseDir, 'dark', `${file.rel}.html`)
      mkdirSync(dirname(lightFile), { recursive: true })
      mkdirSync(dirname(darkFile), { recursive: true })
      writeFileSync(lightFile, `${outputs[LIGHT_THEME]}\n`)
      writeFileSync(darkFile, `${outputs[DARK_THEME]}\n`)
      manifest.push({ path: file.rel, size: file.size })
    }
    writeFileSync(join(caseDir, 'files.json'), `${JSON.stringify(manifest, null, 2)}\n`)
    return manifest.length
  } finally {
    highlighter.dispose()
  }
}
