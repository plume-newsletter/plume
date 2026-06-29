import { render, screen, waitFor, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { ReportPage } from '@/features/campaigns/ReportPage'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

const sampleReport = {
  recipients: 3,
  sent: 2,
  opens: { total: 3, unique: 2 },
  clicks: { total: 1, unique: 1 },
  bounces: 1,
  complaints: 0,
  unsubscribes: 1,
}

function wrap(path = '/campaigns/c1/report') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/campaigns/:id/report" element={<ReportPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/** Walk up from a label element to its enclosing card (data-slot="card"). */
function cardOf(labelEl: HTMLElement): HTMLElement {
  const card = labelEl.closest('[data-slot="card"]')
  if (!card) throw new Error(`No card ancestor found for label "${labelEl.textContent}"`)
  return card as HTMLElement
}

test('renders stat tiles from GET /api/campaigns/c1/report', async () => {
  server.use(
    http.get('/api/campaigns/c1/report', () => HttpResponse.json(sampleReport)),
  )

  wrap()

  await waitFor(() => expect(screen.getByText('Delivered')).toBeInTheDocument())

  // Delivered = recipients(3) - bounces(1) = 2
  expect(within(cardOf(screen.getByText('Delivered'))).getByText('2')).toBeInTheDocument()
  // Unique opens → 2
  expect(within(cardOf(screen.getByText('Unique opens'))).getByText('2')).toBeInTheDocument()
  // Clicks → 1
  expect(within(cardOf(screen.getByText('Clicks'))).getByText('1')).toBeInTheDocument()
  // Bounced → 1
  expect(within(cardOf(screen.getByText('Bounced'))).getByText('1')).toBeInTheDocument()
  // Pure operating cost + engagement chart present
  expect(screen.getByText('SES cost')).toBeInTheDocument()
  expect(screen.getByText('Engagement over 48 hours')).toBeInTheDocument()
})

test('shows "No report yet." on 404', async () => {
  server.use(
    http.get('/api/campaigns/c1/report', () => HttpResponse.json({ error: 'not found' }, { status: 404 })),
  )

  wrap()

  await waitFor(() => expect(screen.getByText('No report yet.')).toBeInTheDocument())
})
