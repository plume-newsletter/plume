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

const sampleCampaign = {
  id: 'c1',
  owner_id: 'o',
  brand_id: 'b1',
  subject: 'Hello',
  html_body: '<p>Hi</p>',
  plain_body: 'Hi',
  body_json: '[]',
  status: 'draft',
  scheduled_at: null,
  created_at: '',
}

function wrap(path = '/campaigns/c1') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/campaigns/:id" element={<ComposePage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

test('loads campaign: subject input value is populated', async () => {
  server.use(
    http.get('/api/campaigns/c1', () => HttpResponse.json(sampleCampaign)),
    http.get('/api/lists', () => HttpResponse.json([])),
    http.post('/api/blocks/render', () => HttpResponse.json({ html: '<p>Hi</p>' })),
  )

  wrap()

  // Wait for the subject input to appear and have the correct value
  await waitFor(() => {
    const input = screen.getByLabelText('Subject') as HTMLInputElement
    expect(input.value).toBe('Hello')
  })
})

// NOTE: 'typing into HTML textarea updates iframe srcdoc live' was removed because the
// HTML Body and Plain Text Body textareas were replaced by the block builder (Task 5).
// The preview iframe is now driven by the /api/blocks/render API, not a direct textarea binding.
// Preview coverage will be added in Task 6.

test('editing the subject auto-saves a PUT with subject and bodyJson', async () => {
  let capturedBody: unknown

  server.use(
    http.get('/api/campaigns/c1', () => HttpResponse.json(sampleCampaign)),
    http.get('/api/lists', () => HttpResponse.json([])),
    http.post('/api/blocks/render', () => HttpResponse.json({ html: '<p>Hi</p>' })),
    http.put('/api/campaigns/c1', async ({ request }) => {
      capturedBody = await request.json()
      return HttpResponse.json({ ...sampleCampaign, subject: 'Hello!' })
    }),
  )

  wrap()

  // Wait for form to load
  await waitFor(() => {
    const input = screen.getByLabelText('Subject') as HTMLInputElement
    expect(input.value).toBe('Hello')
  })

  // Edit subject → debounced auto-save fires a PUT (legacy html_body seeds an html block)
  const user = userEvent.setup()
  await user.type(screen.getByLabelText('Subject'), '!')

  await waitFor(() => {
    expect((capturedBody as Record<string, unknown>).subject).toBe('Hello!')
    expect((capturedBody as Record<string, unknown>).bodyJson).toContain('"type":"html"')
  })
})
