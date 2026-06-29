import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { ComposePage } from '@/features/campaigns/ComposePage'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

const base = {
  id: 'c1', owner_id: 'o', brand_id: 'b1', subject: 'Hi',
  html_body: '', plain_body: '', body_json: '[]', status: 'draft',
  scheduled_at: null, created_at: '2024-01-01T00:00:00Z',
}

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/campaigns/c1']}>
        <Routes><Route path="/campaigns/:id" element={<ComposePage />} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

test('adds a heading block from the palette and saves bodyJson', async () => {
  let saved: { subject?: string; bodyJson?: string } = {}
  server.use(
    http.get('/api/campaigns/c1', () => HttpResponse.json(base)),
    http.post('/api/blocks/render', () => HttpResponse.json({ html: '<h2>Heading</h2>' })),
    http.put('/api/campaigns/c1', async ({ request }) => {
      saved = (await request.json()) as typeof saved
      return HttpResponse.json({ ...base, body_json: saved.bodyJson })
    }),
  )
  wrap()

  await screen.findByText(/content blocks/i)
  await userEvent.click(screen.getByRole('button', { name: /add heading/i }))

  // Auto-save persists the new block (debounced PUT).
  await waitFor(() => expect(saved.bodyJson).toContain('"type":"heading"'))
})

test('rehydrates blocks from a real body_json string on load', async () => {
  server.use(
    http.get('/api/campaigns/c1', () =>
      HttpResponse.json({ ...base, body_json: '[{"type":"heading","text":"Saved title","level":2}]' })),
    http.post('/api/blocks/render', () => HttpResponse.json({ html: '<h2>Saved title</h2>' })),
  )
  wrap()
  await waitFor(() => expect(screen.getByDisplayValue('Saved title')).toBeInTheDocument())
})

test('legacy campaign with html_body seeds a single html block', async () => {
  server.use(
    http.get('/api/campaigns/c1', () =>
      HttpResponse.json({ ...base, html_body: '<p>old</p>', body_json: '[]' })),
    http.post('/api/blocks/render', () => HttpResponse.json({ html: '<p>old</p>' })),
  )
  wrap()
  // The HTML block's content should be present in an editor/inspector field.
  await waitFor(() => expect(screen.getByDisplayValue('<p>old</p>')).toBeInTheDocument())
})
