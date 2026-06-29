import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { ThemeProvider } from '@/components/theme/ThemeProvider'
import { AppShell } from '@/components/AppShell'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function wrap(initialPath = '/brands') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <ThemeProvider>
        <MemoryRouter initialEntries={[initialPath]}>
          <Routes>
            <Route path="/" element={<AppShell />}>
              <Route index element={<div>dashboard content</div>} />
              <Route path="brands" element={<div>brands content</div>} />
              <Route path="*" element={<div>placeholder content</div>} />
            </Route>
          </Routes>
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

test('renders all nav sections incl. placeholders and a log out control', async () => {
  server.use(http.get('/api/me', () => HttpResponse.json({ email: 'a@plume.test', fullName: 'Ada Lovelace' })))
  wrap()
  await waitFor(() => expect(screen.getByText('a@plume.test')).toBeInTheDocument())
  await waitFor(() => expect(screen.getByText('Ada Lovelace')).toBeInTheDocument())

  // built sections
  expect(screen.getAllByRole('link', { name: /dashboard/i })[0]).toBeInTheDocument()
  expect(screen.getAllByRole('link', { name: /brands/i })[0]).toBeInTheDocument()
  expect(screen.getAllByRole('link', { name: /lists & subscribers/i })[0]).toBeInTheDocument()
  expect(screen.getAllByRole('link', { name: /campaigns/i })[0]).toBeInTheDocument()
  expect(screen.getAllByRole('link', { name: /settings/i })[0]).toBeInTheDocument()
  // placeholders
  expect(screen.getAllByRole('link', { name: /segments/i })[0]).toBeInTheDocument()
  expect(screen.getAllByRole('link', { name: /automations/i })[0]).toBeInTheDocument()
  expect(screen.getAllByRole('link', { name: /deliverability/i })[0]).toBeInTheDocument()
  // group headers
  expect(screen.getAllByText('Overview')[0]).toBeInTheDocument()
  expect(screen.getAllByText('System')[0]).toBeInTheDocument()

  expect(screen.getByRole('button', { name: /log out/i })).toBeInTheDocument()
})

test('active route link has aria-current and brand active styling', async () => {
  server.use(http.get('/api/me', () => HttpResponse.json({ email: 'a@plume.test', fullName: 'Ada Lovelace' })))
  wrap('/brands')
  await waitFor(() => expect(screen.getByText('brands content')).toBeInTheDocument())

  const brandsLinks = screen.getAllByRole('link', { name: /brands/i })
  const activeLink = brandsLinks.find((l) => l.getAttribute('aria-current') === 'page')
  expect(activeLink).toBeDefined()
  expect(activeLink!.className).toMatch(/text-primary-text/)

  const dashboardLinks = screen.getAllByRole('link', { name: /dashboard/i })
  expect(dashboardLinks[0]).not.toHaveAttribute('aria-current', 'page')
})
