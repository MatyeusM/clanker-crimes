import { execFileSync } from 'node:child_process'
import { mkdirSync } from 'node:fs'
import { dirname } from 'node:path'

const ZIP_EXCLUDES = [
  'node_modules/*',
  'target/*',
  'dist/*',
  'build/*',
  '.git/*',
  '.astro/*',
  'coverage/*',
  '*.log',
  'tmp-*/*',
]

// Zips the whole package folder (minus generated dirs) into destZip.
export function zipPackage(pkgDir, destZip) {
  mkdirSync(dirname(destZip), { recursive: true })
  const args = ['-r', '-q', destZip, '.', '-x', ...ZIP_EXCLUDES]
  try {
    execFileSync('zip', args, { cwd: pkgDir, stdio: 'pipe' })
  } catch (error) {
    throw new Error(`zip failed: ${error.message}`)
  }
}
