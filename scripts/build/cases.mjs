import { createHash } from 'node:crypto'
import { writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { isCrimeApplicable } from './files.mjs'
import { shuffle } from './util.mjs'

export const MAX_CHOICES = 24

// Picks up to MAX_CHOICES crimes: all solutions plus random applicable
// distractors (language-prefixed crimes only appear for matching packages).
// Returns { choices, bits } where bits[i] marks choices[i] as a solution.
export function buildChoices({ allCrimes, extSet, solutions }) {
  const solutionSet = new Set(solutions)
  const solutionCrimes = allCrimes.filter(crime => solutionSet.has(crime.key))
  const distractors = shuffle(
    allCrimes.filter(crime => !solutionSet.has(crime.key) && isCrimeApplicable(crime.key, extSet)),
  )
  const choices = shuffle([
    ...solutionCrimes,
    ...distractors.slice(0, MAX_CHOICES - solutionCrimes.length),
  ])
  return { choices, bits: choices.map(crime => solutionSet.has(crime.key)) }
}

export function bitString(bits) {
  return bits.map(bit => (bit ? '1' : '0')).join('')
}

// Solution filename: sha256("<slug><bits>"), e.g. "alpha" + "10110...".
// The frontend recomputes this from the player's selection, so a correct
// verdict transparently resolves while wrong ones 404.
export function solutionHash(slug, bits) {
  return createHash('sha256')
    .update(`${slug}${bitString(bits)}`)
    .digest('hex')
}

// Writes choices.json plus the hashed solution file pointing at the next
// case slug (null after the last case). Returns the solution filename.
export function writeCaseFiles(caseDir, slug, choices, bits, nextSlug) {
  writeFileSync(join(caseDir, 'choices.json'), `${JSON.stringify(choices, null, 2)}\n`)
  const hash = solutionHash(slug, bits)
  const url = nextSlug ?? null
  writeFileSync(join(caseDir, `${hash}.json`), `${JSON.stringify({ url }, null, 2)}\n`)
  return hash
}
