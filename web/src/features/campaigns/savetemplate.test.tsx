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
  body_json: '[{"id":"b1","type":"heading","text":"Hi","level":2}]',
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

test('a "Save as template" button is visible in the builder toolbar', async () => {
  server.use(
    http.get('/api/campaigns/c1', () => HttpResponse.json(sampleCampaign)),
    http.get('/api/lists', () => HttpResponse.json([])),
    http.post('/api/blocks/render', () => HttpResponse.json({ html: '<p>Hi</p>' })),
  )

  wrap()

  await waitFor(() =>
    expect(screen.getByRole('button', { name: /save as template/i })).toBeInTheDocument(),
  )
})

test('clicking "Save as template" opens a dialog with a name input and category select', async () => {
  server.use(
    http.get('/api/campaigns/c1', () => HttpResponse.json(sampleCampaign)),
    http.get('/api/lists', () => HttpResponse.json([])),
    http.post('/api/blocks/render', () => HttpResponse.json({ html: '<p>Hi</p>' })),
  )

  wrap()

  const user = userEvent.setup()

  await waitFor(() =>
    expect(screen.getByRole('button', { name: /save as template/i })).toBeInTheDocument(),
  )
  await user.click(screen.getByRole('button', { name: /save as template/i }))

  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
  expect(screen.getByRole('textbox', { name: /name/i })).toBeInTheDocument()
  expect(screen.getByRole('combobox')).toBeInTheDocument()
})

test('submitting the dialog POSTs /api/templates with { name, category, bodyJson }', async () => {
  let captured: unknown

  server.use(
    http.get('/api/campaigns/c1', () => HttpResponse.json(sampleCampaign)),
    http.get('/api/lists', () => HttpResponse.json([])),
    http.post('/api/blocks/render', () => HttpResponse.json({ html: '<p>Hi</p>' })),
    http.post('/api/templates', async ({ request }) => {
      captured = await request.json()
      const body = captured as { name: string; category: string; bodyJson: unknown[] }
      return HttpResponse.json({
        id: 't1',
        name: body.name,
        category: body.category,
        bodyJson: body.bodyJson,
        prebuilt: false,
        createdAt: '',
      })
    }),
  )

  wrap()

  const user = userEvent.setup()

  await waitFor(() =>
    expect(screen.getByRole('button', { name: /save as template/i })).toBeInTheDocument(),
  )
  await user.click(screen.getByRole('button', { name: /save as template/i }))

  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())

  // Clear and type a custom name
  const nameInput = screen.getByRole('textbox', { name: /name/i })
  await user.clear(nameInput)
  await user.type(nameInput, 'My Test Template')

  // Submit using the dialog footer button
  await user.click(screen.getByRole('button', { name: 'Save template' }))

  await waitFor(() => {
    const body = captured as Record<string, unknown>
    expect(body.name).toBe('My Test Template')
    expect(body.category).toBe('Newsletter')
    expect(Array.isArray(body.bodyJson)).toBe(true)
    expect((body.bodyJson as unknown[]).length).toBeGreaterThanOrEqual(1)
  })
})
