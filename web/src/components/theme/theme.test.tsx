import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, test } from 'vitest'
import { ThemeProvider, useTheme } from './ThemeProvider'

function Probe() {
  const { theme, toggle } = useTheme()
  return <button onClick={toggle}>theme:{theme}</button>
}

beforeEach(() => {
  localStorage.clear()
  document.documentElement.classList.remove('dark')
})

test('toggles dark class and persists', async () => {
  render(<ThemeProvider><Probe /></ThemeProvider>)
  const btn = screen.getByRole('button')
  // default resolves to light in jsdom (no prefers-color-scheme)
  expect(document.documentElement.classList.contains('dark')).toBe(false)
  await userEvent.click(btn)
  expect(document.documentElement.classList.contains('dark')).toBe(true)
  expect(localStorage.getItem('plume-theme')).toBe('dark')
})
