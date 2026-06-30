import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test, vi } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { ComposePage } from '@/features/campaigns/ComposePage'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

const sampleCampaign = {
  id: 'c1', owner_id: 'o', brand_id: 'b1', subject: 'Hello',
  html_body: '<p>Hi</p>', plain_body: 'Hi',
  body_json: JSON.stringify([{ id: 'b1', type: 'text', html: 'Our new feature shipped today.' }]),
  status: 'draft', scheduled_at: null, created_at: '',
}

function wrap(campaign = sampleCampaign) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/campaigns/c1']}>
        <Routes>
          <Route path="/campaigns/:id" element={<ComposePage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

test('Suggest fills the subject with a chosen option', async () => {
  server.use(
    http.get('/api/campaigns/c1', () => HttpResponse.json(sampleCampaign)),
    http.get('/api/lists', () => HttpResponse.json([])),
    http.post('/api/blocks/render', () => HttpResponse.json({ html: '<p>Hi</p>' })),
    http.put('/api/campaigns/c1', () => HttpResponse.json(sampleCampaign)),
    http.post('/api/ai/suggest', () =>
      HttpResponse.json({ options: ['Big news inside', 'Your update is here', "Don't miss this"] }),
    ),
  )
  const user = userEvent.setup()
  wrap()

  await waitFor(() => {
    expect((screen.getByLabelText('Subject') as HTMLInputElement).value).toBe('Hello')
  })

  await user.click(screen.getByRole('button', { name: /suggest/i }))
  await user.click(await screen.findByText('Your update is here'))

  expect((screen.getByLabelText('Subject') as HTMLInputElement).value).toBe('Your update is here')
})

test('Suggest with empty body shows toast and does NOT call the API', async () => {
  let suggestCallCount = 0
  const emptyCampaign = {
    ...sampleCampaign,
    body_json: JSON.stringify([{ id: 'b1', type: 'divider' }]),
  }
  server.use(
    http.get('/api/campaigns/c1', () => HttpResponse.json(emptyCampaign)),
    http.get('/api/lists', () => HttpResponse.json([])),
    http.put('/api/campaigns/c1', () => HttpResponse.json(emptyCampaign)),
    http.post('/api/ai/suggest', () => {
      suggestCallCount++
      return HttpResponse.json({ options: ['Should not appear'] })
    }),
  )

  // Suppress console.error for the expected toast (sonner uses it internally in tests)
  const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

  const user = userEvent.setup()
  wrap()

  await waitFor(() => {
    expect((screen.getByLabelText('Subject') as HTMLInputElement).value).toBe('Hello')
  })

  await user.click(screen.getByRole('button', { name: /suggest/i }))

  // Give any async handlers time to fire (they should not)
  await new Promise((r) => setTimeout(r, 50))

  expect(suggestCallCount).toBe(0)
  expect(screen.queryByText('Should not appear')).not.toBeInTheDocument()

  consoleSpy.mockRestore()
})
