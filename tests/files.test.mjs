import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import assert from 'node:assert/strict'
import test from 'node:test'
import {
  extensionOf,
  highlightable,
  isCrimeApplicable,
  isListedBinary,
  walkPackage,
} from '../scripts/build/files.mjs'

// Spoiler rule: these tests only use synthetic fixtures. Never import
// .env.local.json or assert on real slugs, hashes, or solution files —
// CI logs must stay spoiler-free.

test('extensionOf lowercases and handles missing extensions', () => {
  assert.equal(extensionOf('Font.TTF'), 'ttf')
  assert.equal(extensionOf('archive.tar.gz'), 'gz')
  assert.equal(extensionOf('README'), '')
})

test('isCrimeApplicable: unprefixed crimes apply everywhere', () => {
  assert.equal(isCrimeApplicable('godfile', new Set()), true)
  assert.equal(isCrimeApplicable('godfile', new Set(['rs'])), true)
})

test('isCrimeApplicable: prefixed crimes need a matching extension', () => {
  assert.equal(isCrimeApplicable('ts:any-spam', new Set(['ts', 'tsx'])), true)
  assert.equal(isCrimeApplicable('ts:any-spam', new Set(['rs'])), false)
  assert.equal(isCrimeApplicable('TS:any-spam', new Set(['ts'])), true)
})

test('isCrimeApplicable: unknown prefixes match literally', () => {
  assert.equal(isCrimeApplicable('lua:openlibs-all', new Set(['lua'])), true)
  assert.equal(isCrimeApplicable('lua:openlibs-all', new Set(['js'])), false)
})

test('isListedBinary flags fonts and media, never lockfiles', () => {
  const file = (rel, ext) => ({ rel, ext, abs: rel, size: 1 })
  assert.equal(isListedBinary(file('fonts/a.ttf', 'ttf')), true)
  assert.equal(isListedBinary(file('img/hero.png', 'png')), true)
  assert.equal(isListedBinary(file('src/main.rs', 'rs')), false)
  assert.equal(isListedBinary(file('Cargo.lock', 'lock')), false)
  assert.equal(isListedBinary(file('uv.lock', 'lock')), false)
  assert.equal(isListedBinary(file('pnpm-lock.yaml', 'yaml')), false)
})

test('highlightable drops lockfiles, binaries, and oversize files', () => {
  const file = (rel, ext, size) => ({ rel, ext, abs: rel, size })
  const files = [
    file('src/main.rs', 'rs', 100),
    file('Cargo.lock', 'lock', 100),
    file('fonts/a.ttf', 'ttf', 100),
    file('src/huge.rs', 'rs', 256 * 1024 + 1),
  ]
  assert.deepEqual(
    highlightable(files).map(entry => entry.rel),
    ['src/main.rs'],
  )
})

test('walkPackage sorts, skips junk dirs, and records extensions', () => {
  const root = mkdtempSync(join(tmpdir(), 'cc-walk-'))
  mkdirSync(join(root, 'src', 'nested'), { recursive: true })
  mkdirSync(join(root, 'node_modules'), { recursive: true })
  mkdirSync(join(root, 'tmp-scratch'), { recursive: true })
  writeFileSync(join(root, 'b.rs'), 'fn main() {}')
  writeFileSync(join(root, 'src', 'a.rs'), 'x')
  writeFileSync(join(root, 'src', 'nested', 'c.rs'), 'y')
  writeFileSync(join(root, 'node_modules', 'skip.js'), 'z')
  writeFileSync(join(root, 'tmp-scratch', 'skip.js'), 'z')

  const files = walkPackage(join(root))
  assert.deepEqual(
    files.map(entry => entry.rel),
    ['b.rs', 'src/a.rs', 'src/nested/c.rs'],
  )
  assert.ok(files.every(entry => entry.ext === 'rs' && entry.size > 0))
})
