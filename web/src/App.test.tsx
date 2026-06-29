import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AppShell } from '@/components/AppShell'
import { ThemeProvider } from '@/components/theme/ThemeProvider'
import { MemoryRouter, Routes, Route } from 'react-router-dom'

test('app shell renders nav links', () => {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <ThemeProvider>
        <MemoryRouter>
          <Routes>
            <Route path="/" element={<AppShell />}>
              <Route index element={<div>home</div>} />
            </Route>
          </Routes>
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  )
  expect(screen.getAllByText('Plume')[0]).toBeInTheDocument()
  expect(screen.getAllByText('Campaigns')[0]).toBeInTheDocument()
})
