import { readFileSync } from 'node:fs'
import { dirname } from 'node:path'
import { expect, test } from 'vitest'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)
const css = readFileSync(`${__dirname}/index.css`, 'utf8')

const VARS = [
  '--surface-2', '--surface-3', '--faint', '--border-strong',
  '--primary-weak', '--primary-text', '--amber', '--amber-weak',
  '--success-weak', '--danger', '--danger-weak', '--purple', '--purple-weak',
  '--sidebar-active', '--shadow', '--code-bg', '--code-fg',
]
const THEME = [
  '--color-surface-2', '--color-faint', '--color-primary-weak',
  '--color-primary-text', '--color-amber-weak', '--color-sidebar-active',
  '--color-border-strong',
]

test('index.css defines the ported prototype tokens in light and dark', () => {
  for (const v of VARS) {
    // present in :root and in .dark
    expect(css.split(':root')[1] ?? '').toContain(v)
    expect(css.split('.dark')[1] ?? '').toContain(v)
  }
})

test('index.css exposes the new tokens as Tailwind theme colors', () => {
  for (const c of THEME) expect(css).toContain(c)
})
