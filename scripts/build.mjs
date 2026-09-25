// Case builder: turns the packages configured in .env.local.json into
// playable cases under dist/case/<slug>/.
//
// Per case it produces:
//   light/<path>.html, dark/<path>.html  shiki-highlighted source (ayu themes)
//   files.json                            manifest of listed files
//                                         ({ path, size, binary? })
//   artifact.zip                          the full package for inspection
//   preview.html                          single-file vite build (vite only)
//   meta.json                             { hasPreview } so the frontend never
//                                         has to probe for preview.html
//   choices.json                          up to 24 crimes, solutions mixed in
//   <sha256(slug + bits)>.json           { url: <next slug> | null }
//
// The hashed solution file lets the app advance blindly: the frontend hashes
// the player's selection and fetches that file. Correct verdicts resolve to
// the next case, wrong ones 404.
import { mkdirSync, rmSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { zipPackage } from './build/archive.mjs'
import {
  loadConfig,
  loadCrimes,
  validateCrimeReferences,
  validatePackageDirs,
} from './build/config.mjs'
import { buildChoices, writeCaseFiles } from './build/cases.mjs'
import { packageExtensions, walkPackage } from './build/files.mjs'
import { highlightPackage } from './build/highlight.mjs'
import { buildPreview } from './build/preview.mjs'
import { assignCases, loadWordlist } from './build/slugs.mjs'

const ROOT = resolve(import.meta.dirname, '..')
const PACKAGES_DIR = resolve(ROOT, 'packages')
const CASES_DIR = resolve(ROOT, 'dist/case')

// Pass --debug to also print generated slugs and solution filenames.
// Default output only mentions the original package names, so CI logs stay
// spoiler-free when .env.local.json comes from a secret.
const DEBUG = process.argv.includes('--debug')

function debug(message) {
  if (DEBUG) console.log(`  [debug] ${message}`)
}

async function main() {
  // 1. Load and validate .env.local.json against packages/ and data.json.
  const entries = loadConfig(ROOT)
  validatePackageDirs(entries, PACKAGES_DIR)
  const crimes = loadCrimes(ROOT)
  validateCrimeReferences(entries, crimes)

  // 2. Assign one random slug per package; "alpha" is always first.
  const words = loadWordlist(resolve(ROOT, 'scripts/english.txt'))
  const cases = assignCases(entries, words)

  // 3. Rebuild dist/case from scratch so stale cases never linger.
  rmSync(CASES_DIR, { recursive: true, force: true })
  mkdirSync(CASES_DIR, { recursive: true })

  for (const [index, { pkg, crimes: solutions, slug }] of cases.entries()) {
    const pkgDir = resolve(PACKAGES_DIR, pkg)
    const caseDir = resolve(CASES_DIR, slug)
    mkdirSync(caseDir, { recursive: true })
    console.log(`Case ${index + 1}/${cases.length}: ${pkg}`)
    debug(`slug: ${slug}`)

    const files = walkPackage(pkgDir)
    const { highlighted, listed } = await highlightPackage(pkgDir, caseDir, files)
    console.log(`  listed ${listed} files (${highlighted} highlighted)`)

    zipPackage(pkgDir, resolve(caseDir, 'artifact.zip'))
    console.log('  wrote artifact.zip')

    const hasPreview = buildPreview({ pkgDir, pkg, rootDir: ROOT, caseDir })
    console.log(hasPreview ? '  wrote preview.html' : '  no preview (not a vite project)')
    writeFileSync(resolve(caseDir, 'meta.json'), `${JSON.stringify({ hasPreview })}\n`)

    const nextSlug = cases[index + 1]?.slug ?? null
    const { choices, bits } = buildChoices({
      allCrimes: crimes,
      extSet: packageExtensions(files),
      solutions,
    })
    const hashFile = writeCaseFiles(caseDir, slug, choices, bits, nextSlug)
    console.log(`  wrote choices.json (${choices.length} choices) and solution file`)
    debug(`solution file: ${hashFile}.json`)
    debug(`next: ${nextSlug ?? '(victory)'}`)
  }

  console.log(`Done: ${cases.length} case(s) in dist/case/.`)
}

await main()
