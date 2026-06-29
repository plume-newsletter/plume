import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { ABTestsPage } from '@/features/ab-tests/ABTestsPage'

beforeAll(() => server.listen()); afterEach(() => server.resetHandlers()); afterAll(() => server.close())

const test1 = { id: 't1', campaignId: 'c1', listId: 'l1', subjectA: 'Subject A 🚀', subjectB: 'Subject B', testPercent: 20, status: 'running', winner: '', createdAt: '' }

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}><MemoryRouter><ABTestsPage /></MemoryRouter></QueryClientProvider>)
}

test('renders a running test with both variants and the winning badge', async () => {
  server.use(
    http.get('/api/ab-tests', () => HttpResponse.json([test1])),
    http.get('/api/campaigns', () => HttpResponse.json([{ id: 'c1', owner_id: 'o', brand_id: 'b1', subject: 'Spring update', status: 'draft', html_body: '', plain_body: '', body_json: '[]', scheduled_at: null, created_at: '' }])),
    http.get('/api/lists', () => HttpResponse.json([{ id: 'l1', owner_id: 'o', brand_id: 'b1', name: 'Main', created_at: '' }])),
    http.get('/api/ab-tests/t1/results', () => HttpResponse.json({ status: 'running', winner: '', variants: [
      { variant: 'a', subject: 'Subject A 🚀', sent: 100, openRate: 0.61, clickRate: 0.11 },
      { variant: 'b', subject: 'Subject B', sent: 100, openRate: 0.54, clickRate: 0.09 },
    ] })),
  )
  wrap()
  await waitFor(() => expect(screen.getByText('Subject A 🚀')).toBeInTheDocument())
  expect(screen.getByText('Subject B')).toBeInTheDocument()
  expect(screen.getByText('61.0%')).toBeInTheDocument()
  // WINNING badge appears once, on the leader (variant A)
  expect(screen.getByText(/winning/i)).toBeInTheDocument()
})
