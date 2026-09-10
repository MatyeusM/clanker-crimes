import { readdirSync, statSync } from 'node:fs'
import { extname, join, relative, sep } from 'node:path'

const SKIP_DIRS = new Set(['node_modules', 'target', 'dist', 'build', '.git', '.astro', 'coverage'])

// Lockfiles are noise for highlighting but stay in the zip artifact.
const SKIP_FILES = new Set(['pnpm-lock.yaml', 'package-lock.json', 'yarn.lock', 'Cargo.lock'])

// Binary / font / media extensions never highlighted.
const SKIP_EXTS = new Set([
  'png',
  'jpg',
  'jpeg',
  'gif',
  'webp',
  'ico',
  'svg',
  'bmp',
  'avif',
  'mp3',
  'mp4',
  'pdf',
  'wasm',
  'woff',
  'woff2',
  'ttf',
  'otf',
  'eot',
  'node',
  'exe',
  'dll',
  'so',
  'dylib',
  'bin',
  'dat',
  'lock',
])

export const MAX_HIGHLIGHT_BYTES = 256 * 1024

// Maps a ("ext:"-style) crime prefix to the file extensions it applies to.
// Unknown prefixes are treated as a literal file extension.
export const LANG_ALIASES = {
  ts: ['ts', 'tsx', 'mts', 'cts'],
  js: ['js', 'mjs', 'cjs', 'jsx'],
  rs: ['rs'],
  css: ['css'],
  html: ['html'],
  json: ['json'],
  md: ['md'],
  toml: ['toml'],
}

export function extensionOf(fileName) {
  return extname(fileName).slice(1).toLowerCase()
}

function toPosix(path) {
  return path.split(sep).join('/')
}

function isSkippedDir(name) {
  return SKIP_DIRS.has(name) || name.startsWith('tmp-')
}

export function walkPackage(pkgDir) {
  const files = []
  const visit = dir => {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      if (entry.isDirectory()) {
        if (!isSkippedDir(entry.name)) visit(join(dir, entry.name))
      } else if (entry.isFile()) {
        const abs = join(dir, entry.name)
        files.push({
          rel: toPosix(relative(pkgDir, abs)),
          abs,
          ext: extensionOf(entry.name),
          size: statSync(abs).size,
        })
      }
    }
  }
  visit(pkgDir)
  files.sort((a, b) => (a.rel < b.rel ? -1 : 1))
  return files
}

export function packageExtensions(files) {
  return new Set(files.map(file => file.ext).filter(Boolean))
}

export function highlightable(files) {
  return files.filter(
    file =>
      !SKIP_FILES.has(file.rel.split('/').pop()) &&
      !SKIP_EXTS.has(file.ext) &&
      file.size <= MAX_HIGHLIGHT_BYTES,
  )
}

// A crime key is optionally prefixed with "ext:" (e.g. "ts:any-spam").
// Unprefixed crimes apply to every package; prefixed ones only to packages
// containing a matching file extension.
export function isCrimeApplicable(crimeKey, extSet) {
  const separator = crimeKey.indexOf(':')
  if (separator === -1) return true
  const prefix = crimeKey.slice(0, separator).toLowerCase()
  const wanted = LANG_ALIASES[prefix] ?? [prefix]
  return wanted.some(ext => extSet.has(ext))
}

export function escapeHtml(text) {
  return text
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
}
