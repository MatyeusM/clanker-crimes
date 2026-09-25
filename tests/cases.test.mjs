import assert from 'node:assert/strict'
import test from 'node:test'
import { MAX_CHOICES, bitString, buildChoices, solutionHash } from '../scripts/build/cases.mjs'

// Spoiler rule: synthetic fixture crimes only. Never log or assert on real
// slugs, hashes, or solution files — CI logs must stay spoiler-free.

const fixtureCrimes = [
  { key: 'alpha-crime', name: 'Alpha', description: 'First fixture crime.' },
  { key: 'beta-crime', name: 'Beta', description: 'Second fixture crime.' },
  { key: 'ts:gamma', name: 'Gamma', description: 'TypeScript-only fixture crime.' },
  { key: 'rs:delta', name: 'Delta', description: 'Rust-only fixture crime.' },
]

test('bitString joins bits without separators', () => {
  assert.equal(bitString([true, false, true]), '101')
  assert.equal(bitString([]), '')
})

test('solutionHash is stable hex and sensitive to every input', () => {
  const a = solutionHash('some-slug', [true, false])
  const b = solutionHash('some-slug', [true, false])
  assert.equal(a, b)
  assert.match(a, /^[0-9a-f]{64}$/)
  assert.notEqual(a, solutionHash('some-slug', [true, true]))
  assert.notEqual(a, solutionHash('other-slug', [true, false]))
})

test('buildChoices always includes every solution', () => {
  const { choices, bits } = buildChoices({
    allCrimes: fixtureCrimes,
    extSet: new Set(['rs']),
    solutions: ['alpha-crime', 'rs:delta'],
  })
  assert.equal(choices.length, bits.length)
  for (const solution of ['alpha-crime', 'rs:delta']) {
    const index = choices.findIndex(choice => choice.key === solution)
    assert.notEqual(index, -1)
    assert.equal(bits[index], true)
  }
})

test('buildChoices only offers applicable distractors', () => {
  const { choices } = buildChoices({
    allCrimes: fixtureCrimes,
    extSet: new Set(['rs']),
    solutions: ['alpha-crime'],
  })
  const keys = choices.map(choice => choice.key)
  assert.ok(!keys.includes('ts:gamma'))
})

test('buildChoices caps the catalogue at MAX_CHOICES', () => {
  assert.equal(MAX_CHOICES, 24)
  const many = Array.from({ length: 40 }, (_, index) => ({
    key: `crime-${index}`,
    name: `Crime ${index}`,
    description: 'Fixture crime.',
  }))
  const { choices, bits } = buildChoices({
    allCrimes: many,
    extSet: new Set(),
    solutions: ['crime-0', 'crime-1'],
  })
  assert.equal(choices.length, MAX_CHOICES)
  assert.equal(bits.filter(Boolean).length, 2)
})
