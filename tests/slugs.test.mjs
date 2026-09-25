import assert from 'node:assert/strict'
import test from 'node:test'
import { FIRST_SLUG, WORDS_PER_SLUG, assignCases, loadWordlist } from '../scripts/build/slugs.mjs'

// Spoiler rule: synthetic entries only. Never log generated slugs from real
// builds — CI logs must stay spoiler-free.

const entries = [{ pkg: 'one' }, { pkg: 'two' }, { pkg: 'three' }]
const words = Array.from({ length: 64 }, (_, index) => `word${String(index).padStart(2, '0')}`)

test('assignCases keeps every entry and puts alpha first', () => {
  const cases = assignCases(entries, words)
  assert.equal(cases.length, entries.length)
  assert.equal(cases[0].slug, FIRST_SLUG)
  assert.deepEqual(cases.map(entry => entry.pkg).sort(), ['one', 'three', 'two'])
})

test('assignCases generates unique well-formed slugs', () => {
  const cases = assignCases(entries, words)
  const slugs = new Set(cases.map(entry => entry.slug))
  assert.equal(slugs.size, cases.length)
  for (const slug of [...slugs].slice(1)) {
    const parts = slug.split('-')
    assert.equal(parts.length, WORDS_PER_SLUG)
    assert.ok(parts.every(word => /^[a-z0-9]{4,8}$/.test(word)))
  }
})

test('the shipped wordlist yields enough usable words', () => {
  const loaded = loadWordlist(new URL('../scripts/english.txt', import.meta.url))
  assert.ok(loaded.length >= 1000)
})
