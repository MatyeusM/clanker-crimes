import { existsSync, readFileSync, realpathSync, statSync } from 'node:fs'
import { dirname, isAbsolute, relative, resolve, sep } from 'node:path'
import { fail } from './util.mjs'

const KEY_PATTERN = /^([A-Za-z0-9-]+:)?[A-Za-z0-9-]+$/

function loadJson(file, description) {
  if (!existsSync(file)) {
    fail(`${description} does not exist: ${file}`)
  }
  try {
    return JSON.parse(readFileSync(file, 'utf8'))
  } catch (error) {
    fail(`could not parse ${description}: ${error.message}`)
  }
}

// Accepts both the array form [{ pkg, crimes }] and the legacy object form
// { "<pkg>": ["<crime>"] }. Normalizes to [{ pkg, crimes }].
export function loadConfig(root) {
  const file = resolve(root, '.env.local.json')
  const raw = loadJson(file, '.env.local.json')

  const entries = Array.isArray(raw)
    ? raw
    : typeof raw === 'object' && raw !== null
      ? Object.entries(raw).map(([pkg, crimes]) => ({ pkg, crimes }))
      : fail('.env.local.json must contain an array or an object')

  if (entries.length === 0) {
    fail('.env.local.json does not configure any packages')
  }

  const seen = new Set()
  for (const [index, entry] of entries.entries()) {
    if (typeof entry !== 'object' || entry === null || Array.isArray(entry)) {
      fail(`.env.local.json entry ${index} must be an object`)
    }
    if (typeof entry.pkg !== 'string' || entry.pkg.length === 0) {
      fail(`.env.local.json entry ${index} has an invalid pkg`)
    }
    if (seen.has(entry.pkg)) {
      fail(`.env.local.json configures package "${entry.pkg}" twice`)
    }
    seen.add(entry.pkg)
    if (!Array.isArray(entry.crimes) || entry.crimes.length === 0) {
      fail(`configuration for package "${entry.pkg}" must be a non-empty array of crime keys`)
    }
    for (const crimeKey of entry.crimes) {
      if (typeof crimeKey !== 'string' || !KEY_PATTERN.test(crimeKey)) {
        fail(
          `crime keys for package "${entry.pkg}" must all be strings like "godfile" or "ts:any-spam"`,
        )
      }
    }
  }

  return entries
}

// Normalizes { key, label | name, description } to { key, name, description }.
export function loadCrimes(root) {
  const data = loadJson(resolve(root, 'src/data.json'), 'src/data.json')

  if (!Array.isArray(data)) {
    fail('src/data.json must contain an array')
  }

  const keys = new Set()
  const crimes = data.map((entry, index) => {
    if (typeof entry !== 'object' || entry === null || Array.isArray(entry)) {
      fail(`src/data.json entry ${index} must be an object`)
    }
    if (typeof entry.key !== 'string' || !KEY_PATTERN.test(entry.key)) {
      fail(`src/data.json entry ${index} has an invalid key`)
    }
    const name = entry.name ?? entry.label
    if (typeof name !== 'string' || name.length === 0) {
      fail(`src/data.json entry ${index} has an invalid name`)
    }
    if (typeof entry.description !== 'string' || entry.description.length === 0) {
      fail(`src/data.json entry ${index} has an invalid description`)
    }
    if (keys.has(entry.key)) {
      fail(`src/data.json contains duplicate key "${entry.key}"`)
    }
    keys.add(entry.key)
    return { key: entry.key, name, description: entry.description }
  })

  return crimes
}

export function validatePackageDirs(entries, packagesDir) {
  let packagesRoot
  try {
    packagesRoot = realpathSync(packagesDir)
  } catch {
    fail('packages/ does not exist')
  }

  for (const { pkg } of entries) {
    if (
      typeof pkg !== 'string' ||
      pkg.length === 0 ||
      pkg === '.' ||
      pkg === '..' ||
      pkg.includes('/') ||
      pkg.includes('\\')
    ) {
      fail(`configured package "${pkg}" is not a direct packages/ directory`)
    }

    const candidate = resolve(packagesRoot, pkg)
    if (dirname(candidate) !== packagesRoot) {
      fail(`configured package "${pkg}" is not a direct packages/ directory`)
    }

    let realCandidate
    try {
      realCandidate = realpathSync(candidate)
    } catch {
      fail(`configured package "${pkg}" does not exist in packages/`)
    }

    let stats
    try {
      stats = statSync(realCandidate)
    } catch {
      fail(`configured package "${pkg}" does not exist in packages/`)
    }
    const packagePath = relative(packagesRoot, realCandidate)
    if (
      !stats.isDirectory() ||
      !packagePath ||
      packagePath === '..' ||
      packagePath.startsWith(`..${sep}`) ||
      isAbsolute(packagePath) ||
      packagePath.includes(sep)
    ) {
      fail(`configured package "${pkg}" is not a direct packages/ directory`)
    }
  }
}

export function validateCrimeReferences(entries, crimes) {
  const known = new Set(crimes.map(crime => crime.key))
  for (const { pkg, crimes: configured } of entries) {
    for (const crimeKey of configured) {
      if (!known.has(crimeKey)) {
        fail(`package "${pkg}" references an unknown crime key`)
      }
    }
    if (configured.length > 24) {
      fail(`package "${pkg}" configures more than 24 solution crimes`)
    }
  }
}
