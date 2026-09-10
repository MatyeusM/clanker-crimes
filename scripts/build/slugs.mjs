import { readFileSync } from 'node:fs'
import { fail, randomPick, shuffle } from './util.mjs'

export const WORDS_PER_SLUG = 4
export const FIRST_SLUG = 'alpha'

// Only slug-safe words: lowercase ascii, short enough to read well in a URL.
const WORD_PATTERN = /^[a-z]{4,8}$/

export function loadWordlist(file) {
  let text
  try {
    text = readFileSync(file, 'utf8')
  } catch (error) {
    fail(`could not read wordlist: ${error.message}`)
  }
  const words = [...new Set(text.split(/\r?\n/).filter(word => WORD_PATTERN.test(word)))]
  if (words.length < 1000) {
    fail(`wordlist yielded only ${words.length} usable words`)
  }
  return words
}

function generateSlug(words, taken) {
  for (let attempt = 0; attempt < 100; attempt++) {
    const slug = Array.from({ length: WORDS_PER_SLUG }, () => randomPick(words)).join('-')
    if (!taken.has(slug)) {
      taken.add(slug)
      return slug
    }
  }
  fail('could not generate a unique case slug')
}

// Randomly assigns one slug per package. The returned order is the play
// order: shuffled, but the "alpha" case is always first.
export function assignCases(entries, words) {
  const shuffled = shuffle(entries)
  const taken = new Set([FIRST_SLUG])
  const slugs = [FIRST_SLUG]
  for (let i = 1; i < shuffled.length; i++) {
    slugs.push(generateSlug(words, taken))
  }
  return shuffled.map((entry, index) => ({ ...entry, slug: slugs[index] }))
}
