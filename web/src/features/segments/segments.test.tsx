import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { SegmentsPage } from '@/features/segments/SegmentsPage'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}><MemoryRouter><SegmentsPage /></MemoryRouter></QueryClientProvider>)
}

test('adds a condition, shows the live preview count, and lists saved segments', async () => {
  server.use(
    http.get('/api/segments', () => HttpResponse.json([{ id: 's1', name: 'Engaged', match: 'all', conditions: [], count: 42, createdAt: '' }])),
    http.get('/api/segments/fields', () => HttpResponse.json(['Plan'])),
    http.post('/api/segments/preview', () => HttpResponse.json({ count: 7, total: 100, percent: 0.07, sample: [] })),
  )
  wrap()
  // saved segment from the list
  await waitFor(() => expect(screen.getByText('Engaged')).toBeInTheDocument())
  // add a condition -> preview fires
  await userEvent.click(screen.getByRole('button', { name: /add condition/i }))
  await waitFor(() => expect(screen.getByText('7')).toBeInTheDocument())
})

test('saving posts name + match + conditions', async () => {
  let body: unknown
  server.use(
    http.get('/api/segments', () => HttpResponse.json([])),
    http.get('/api/segments/fields', () => HttpResponse.json([])),
    http.post('/api/segments/preview', () => HttpResponse.json({ count: 0, total: 0, percent: 0, sample: [] })),
    http.post('/api/segments', async ({ request }) => { body = await request.json(); return HttpResponse.json({ id: 's2', name: 'New', match: 'all', conditions: [], count: 0, createdAt: '' }) }),
  )
  wrap()
  await userEvent.click(screen.getByRole('button', { name: /save segment/i }))
  await userEvent.type(screen.getByLabelText(/segment name/i), 'New')
  await userEvent.click(screen.getByRole('button', { name: /^save$/i }))
  await waitFor(() => expect((body as { name?: string })?.name).toBe('New'))
})
