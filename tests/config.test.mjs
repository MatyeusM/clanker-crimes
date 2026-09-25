import { spawnSync } from 'node:child_process'
import { mkdtempSync, mkdirSync, symlinkSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import assert from 'node:assert/strict'
import test from 'node:test'
import {
  loadConfig,
  loadCrimes,
  validateCrimeReferences,
  validatePackageDirs,
} from '../scripts/build/config.mjs'

// Spoiler rule: synthetic fixtures only. Failure paths exit the process, so
// they run in a child and only assert on exit codes and fixture key names.

function makeRoot(envLocal, data) {
  const root = mkdtempSync(join(tmpdir(), 'cc-config-'))
  mkdirSync(join(root, 'src'), { recursive: true })
  writeFileSync(join(root, '.env.local.json'), JSON.stringify(envLocal))
  writeFileSync(join(root, 'src', 'data.json'), JSON.stringify(data))
  return root
}

const crimes = [{ key: 'fixture-crime', name: 'Fixture', description: 'Fixture crime.' }]

test('loadConfig accepts the array form', () => {
  const root = makeRoot([{ pkg: 'demo', crimes: ['fixture-crime'] }], crimes)
  assert.deepEqual(loadConfig(root), [{ pkg: 'demo', crimes: ['fixture-crime'] }])
})

test('loadConfig normalizes the legacy object form', () => {
  const root = makeRoot({ demo: ['fixture-crime'] }, crimes)
  assert.deepEqual(loadConfig(root), [{ pkg: 'demo', crimes: ['fixture-crime'] }])
})

test('loadCrimes accepts label as an alias of name', () => {
  const root = makeRoot(
    [{ pkg: 'demo', crimes: ['fixture-crime'] }],
    [{ key: 'fixture-crime', label: 'Fixture', description: 'Fixture crime.' }],
  )
  assert.deepEqual(loadCrimes(root), [
    { key: 'fixture-crime', name: 'Fixture', description: 'Fixture crime.' },
  ])
})

test('package and crime references validate', () => {
  const root = makeRoot([{ pkg: 'demo', crimes: ['fixture-crime'] }], crimes)
  mkdirSync(join(root, 'packages', 'demo'), { recursive: true })
  validatePackageDirs(loadConfig(root), join(root, 'packages'))
  validateCrimeReferences(loadConfig(root), loadCrimes(root))
})

test('package path traversal is rejected', () => {
  const root = makeRoot([{ pkg: '../outside', crimes: ['fixture-crime'] }], crimes)
  const packagesDir = join(root, 'packages')
  mkdirSync(packagesDir, { recursive: true })
  const child = spawnSync(
    process.execPath,
    [
      '--input-type=module',
      '-e',
      `import * as config from ${JSON.stringify(new URL('../scripts/build/config.mjs', import.meta.url).href)};
       config.validatePackageDirs([{ pkg: '../outside' }], ${JSON.stringify(packagesDir)});`,
    ],
    { encoding: 'utf8' },
  )
  assert.equal(child.status, 1)
  assert.match(child.stderr, /direct packages\/ directory/)
})

test('package symlink escapes are rejected', () => {
  const root = makeRoot([{ pkg: 'linked', crimes: ['fixture-crime'] }], crimes)
  const packagesDir = join(root, 'packages')
  const outsideDir = join(root, 'outside')
  mkdirSync(packagesDir, { recursive: true })
  mkdirSync(outsideDir, { recursive: true })
  symlinkSync(outsideDir, join(packagesDir, 'linked'), 'dir')
  const child = spawnSync(
    process.execPath,
    [
      '--input-type=module',
      '-e',
      `import * as config from ${JSON.stringify(new URL('../scripts/build/config.mjs', import.meta.url).href)};
       config.validatePackageDirs([{ pkg: 'linked' }], ${JSON.stringify(packagesDir)});`,
    ],
    { encoding: 'utf8' },
  )
  assert.equal(child.status, 1)
  assert.match(child.stderr, /direct packages\/ directory/)
})

test('unknown crime references fail the build', () => {
  const child = spawnSync(
    process.execPath,
    [
      '--input-type=module',
      '-e',
      `import * as config from ${JSON.stringify(new URL('../scripts/build/config.mjs', import.meta.url).href)};
       config.validateCrimeReferences(
         [{ pkg: 'demo', crimes: ['missing-crime'] }],
         [{ key: 'fixture-crime', name: 'Fixture', description: 'Fixture.' }],
       );`,
    ],
    { encoding: 'utf8' },
  )
  assert.equal(child.status, 1)
  assert.match(child.stderr, /unknown crime key/)
  assert.doesNotMatch(child.stderr, /missing-crime/)
})
