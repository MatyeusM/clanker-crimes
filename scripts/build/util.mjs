import { randomInt } from 'node:crypto'

export function fail(message) {
  console.error(`Build validation failed: ${message}`)
  process.exit(1)
}

export function warn(message) {
  console.warn(`Build warning: ${message}`)
}

// Fisher-Yates shuffle using crypto randomness. Returns a new array.
export function shuffle(items) {
  const result = [...items]
  for (let i = result.length - 1; i > 0; i--) {
    const j = randomInt(i + 1)
    ;[result[i], result[j]] = [result[j], result[i]]
  }
  return result
}

export function randomPick(words) {
  return words[randomInt(words.length)]
}
