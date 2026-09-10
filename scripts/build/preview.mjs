import { execFileSync } from 'node:child_process'
import { cpSync, existsSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { warn } from './util.mjs'

const VITE_CONFIGS = ['vite.config.mts', 'vite.config.mjs', 'vite.config.ts', 'vite.config.js']
const SINGLEFILE_IMPORT = 'import { viteSingleFile } from "vite-plugin-singlefile"'
const BUILD_TIMEOUT_MS = 10 * 60 * 1000

export function findViteConfig(pkgDir) {
  return VITE_CONFIGS.map(name => join(pkgDir, name)).find(file => existsSync(file)) ?? null
}

function patchViteConfig(source) {
  let patched = source.includes('vite-plugin-singlefile')
    ? source
    : `${SINGLEFILE_IMPORT}\n${source}`

  if (/plugins\s*:\s*\[/.test(patched)) {
    return patched.replace(/plugins\s*:\s*\[/, 'plugins: [viteSingleFile(), ')
  }
  if (/defineConfig\(\s*\{/.test(patched)) {
    return patched.replace(/defineConfig\(\s*\{/, 'defineConfig({ plugins: [viteSingleFile()], ')
  }
  throw new Error('could not find a plugins array or defineConfig object to patch')
}

function sh(cmd, args, cwd) {
  execFileSync(cmd, args, { cwd, stdio: 'inherit', timeout: BUILD_TIMEOUT_MS })
}

// Builds a self-contained preview.html for vite projects without touching
// the original package: copy to tmp-<pkg>, inject vite-plugin-singlefile
// (resolved from the repo root), build, move the single file, clean up.
// Returns true when a preview was produced, false otherwise.
export function buildPreview({ pkgDir, pkg, rootDir, caseDir }) {
  const viteConfig = findViteConfig(pkgDir)
  if (!viteConfig) return false

  const safePkg = pkg.replaceAll(/[^a-z0-9-]+/gi, '-')
  const tmpDir = join(rootDir, `tmp-${safePkg}`)
  rmSync(tmpDir, { recursive: true, force: true })

  try {
    cpSync(pkgDir, tmpDir, {
      recursive: true,
      filter: src => {
        const name = src.split('/').pop()
        return (
          !['node_modules', 'dist', 'target', '.git'].includes(name) && !name.startsWith('tmp-')
        )
      },
    })

    const configName = viteConfig.split('/').pop()
    const tmpConfig = join(tmpDir, configName)
    writeFileSync(tmpConfig, patchViteConfig(readFileSync(tmpConfig, 'utf8')))

    sh('pnpm', ['install', '--no-frozen-lockfile'], tmpDir)
    sh('pnpm', ['build'], tmpDir)

    const built = join(tmpDir, 'dist', 'index.html')
    if (!existsSync(built)) throw new Error('vite build produced no dist/index.html')
    cpSync(built, join(caseDir, 'preview.html'))
    return true
  } catch (error) {
    warn(`preview build for "${pkg}" failed, skipping preview.html: ${error.message}`)
    return false
  } finally {
    rmSync(tmpDir, { recursive: true, force: true })
  }
}
