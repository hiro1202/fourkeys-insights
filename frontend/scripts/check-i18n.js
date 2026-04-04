// Build-time i18n key validation
// Checks that EN and JP JSON files have identical key sets.
import { readFileSync } from 'fs'
import { join, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const en = JSON.parse(readFileSync(join(__dirname, '../src/i18n/en.json'), 'utf-8'))
const ja = JSON.parse(readFileSync(join(__dirname, '../src/i18n/ja.json'), 'utf-8'))

const enKeys = new Set(Object.keys(en))
const jaKeys = new Set(Object.keys(ja))

let hasError = false

for (const key of enKeys) {
  if (!jaKeys.has(key)) {
    console.error(`Missing in ja.json: "${key}"`)
    hasError = true
  }
}
for (const key of jaKeys) {
  if (!enKeys.has(key)) {
    console.error(`Missing in en.json: "${key}"`)
    hasError = true
  }
}

if (hasError) {
  console.error('\ni18n key mismatch detected!')
  process.exit(1)
} else {
  console.log(`i18n check passed: ${enKeys.size} keys match`)
}
