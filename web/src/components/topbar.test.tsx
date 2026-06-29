import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { expect, test } from 'vitest'
import { ThemeProvider } from '@/components/theme/ThemeProvider'
import { TopBar } from '@/components/TopBar'

function wrap(path = '/') {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={[path]}>
        <TopBar />
      </MemoryRouter>
    </ThemeProvider>,
  )
}

test('renders the page title, a search box, a theme toggle, and Create', () => {
  wrap('/')
  expect(screen.getByText('Dashboard')).toBeInTheDocument()
  expect(screen.getByPlaceholderText(/search or jump to/i)).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /toggle theme|switch to/i })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /create/i })).toBeInTheDocument()
})

test('derives a breadcrumb title for the campaign editor', () => {
  wrap('/campaigns/abc123')
  expect(screen.getByText(/campaigns \/ editor/i)).toBeInTheDocument()
})

test('theme toggle flips the dark class', async () => {
  wrap('/')
  document.documentElement.classList.remove('dark')
  await userEvent.click(screen.getByRole('button', { name: /toggle theme|switch to/i }))
  expect(document.documentElement.classList.contains('dark')).toBe(true)
})
