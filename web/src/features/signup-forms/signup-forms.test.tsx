import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { SignupFormsPage } from '@/features/signup-forms/SignupFormsPage'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

const sampleList = { id: 'l1', owner_id: 'o', brand_id: 'b1', name: 'Newsletter', created_at: '' }

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}><MemoryRouter><SignupFormsPage /></MemoryRouter></QueryClientProvider>)
}

test('editing heading updates the preview and saving posts the body', async () => {
  let body: unknown
  server.use(
    http.get('/api/signup-forms', () => HttpResponse.json([])),
    http.get('/api/lists', () => HttpResponse.json([sampleList])),
    http.post('/api/signup-forms', async ({ request }) => {
      body = await request.json()
      return HttpResponse.json({ id: 'f1', listId: 'l1', name: 'Hero', heading: 'Join us', description: '', buttonText: 'Subscribe', createdAt: '' })
    }),
  )
  wrap()
  // a new draft exists by default; type a heading
  const heading = await screen.findByLabelText(/heading/i)
  await userEvent.clear(heading)
  await userEvent.type(heading, 'Join us')
  // preview reflects it (the heading text appears in the preview card)
  await waitFor(() => expect(screen.getAllByText('Join us').length).toBeGreaterThan(0))
  // pick the list, give a name, save
  await userEvent.selectOptions(await screen.findByLabelText(/list/i), 'l1')
  await userEvent.type(screen.getByLabelText(/^name/i), 'Hero')
  await userEvent.click(screen.getByRole('button', { name: /save/i }))
  await waitFor(() => expect((body as { heading?: string })?.heading).toBe('Join us'))
})
