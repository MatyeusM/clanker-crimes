// CI helper: writes a throwaway .env.local.json from the public crime
// catalogue (the first three unprefixed crimes for every package) so PR
// checks can run the full case builder without the secret solutions file.
//
// Refuses to overwrite an existing file unless --force is passed, so it can
// never clobber real local solutions by accident. Output stays spoiler-free:
// package names and counts only.
import { existsSync, readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { fail } from './build/util.mjs'

const ROOT = resolve(import.meta.dirname, '..')
const FORCE = process.argv.includes('--force')

const target = resolve(ROOT, '.env.local.json')
if (existsSync(target) && !FORCE) {
  fail('.env.local.json already exists — pass --force to overwrite it')
}

const catalogue = JSON.parse(readFileSync(resolve(ROOT, 'src/data.json'), 'utf8'))
const generic = catalogue.map(entry => entry.key).filter(key => !key.includes(':'))
if (generic.length < 3) {
  fail('src/data.json has fewer than three unprefixed crimes')
}

const packagesDir = resolve(ROOT, 'packages')
const entries = readdirSync(packagesDir)
  .filter(name => statSync(resolve(packagesDir, name)).isDirectory() && !name.startsWith('tmp-'))
  .sort()
  .map(pkg => ({ pkg, crimes: generic.slice(0, 3) }))

if (entries.length === 0) {
  fail('no packages found in packages/')
}

writeFileSync(target, `${JSON.stringify(entries, null, 2)}\n`)
console.log(`wrote throwaway solutions for ${entries.length} package(s)`)
