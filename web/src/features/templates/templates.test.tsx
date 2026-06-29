import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { TemplatesPage } from '@/features/templates/TemplatesPage'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

const prebuilt = [
  { id: 'tpl-1', name: 'Newsletter starter', category: 'Newsletter', bodyJson: [], prebuilt: true, createdAt: '' },
  { id: 'tpl-2', name: 'Product update', category: 'Product', bodyJson: [], prebuilt: true, createdAt: '' },
  { id: 'tpl-3', name: 'Flash sale', category: 'Promo', bodyJson: [], prebuilt: true, createdAt: '' },
]

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <TemplatesPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

test('renders all 3 starter template names', async () => {
  server.use(
    http.get('/api/templates', () => HttpResponse.json(prebuilt)),
    http.get('/api/brands', () => HttpResponse.json([{ id: 'b1', name: 'Acme' }])),
  )

  wrap()

  await waitFor(() => expect(screen.getByText('Newsletter starter')).toBeInTheDocument())
  expect(screen.getByText('Product update')).toBeInTheDocument()
  expect(screen.getByText('Flash sale')).toBeInTheDocument()
})

test('clicking Promo chip filters to show only Promo template', async () => {
  server.use(
    http.get('/api/templates', ({ request }) => {
      const cat = new URL(request.url).searchParams.get('category')
      if (cat === 'Promo') return HttpResponse.json([prebuilt[2]])
      return HttpResponse.json(prebuilt)
    }),
    http.get('/api/brands', () => HttpResponse.json([{ id: 'b1', name: 'Acme' }])),
  )

  wrap()

  // Wait for all templates to appear
  await waitFor(() => expect(screen.getByText('Newsletter starter')).toBeInTheDocument())

  // Click the Promo chip
  const user = userEvent.setup()
  await user.click(screen.getByRole('button', { name: 'Promo' }))

  // Non-Promo template disappears; Promo template remains
  await waitFor(() => expect(screen.queryByText('Newsletter starter')).not.toBeInTheDocument())
  expect(screen.getByText('Flash sale')).toBeInTheDocument()
})

test('clicking a card opens a dialog with brand select and subject input', async () => {
  server.use(
    http.get('/api/templates', () => HttpResponse.json(prebuilt)),
    http.get('/api/brands', () => HttpResponse.json([{ id: 'b1', name: 'Acme' }])),
  )

  wrap()

  await waitFor(() => expect(screen.getByText('Newsletter starter')).toBeInTheDocument())

  const user = userEvent.setup()
  // Each card renders as a button with aria-label = template name
  await user.click(screen.getByRole('button', { name: 'Newsletter starter' }))

  // Dialog should appear with brand select and subject input
  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
  expect(screen.getByLabelText('Brand')).toBeInTheDocument()
  expect(screen.getByLabelText('Subject')).toBeInTheDocument()
})

test('selecting brand + typing subject + submitting calls POST /templates/:id/use', async () => {
  let capturedBody: unknown

  server.use(
    http.get('/api/templates', () => HttpResponse.json(prebuilt)),
    http.get('/api/brands', () => HttpResponse.json([{ id: 'b1', name: 'Acme' }])),
    http.post('/api/templates/:id/use', async ({ request }) => {
      capturedBody = await request.json()
      return HttpResponse.json({ campaignId: 'new1' })
    }),
  )

  wrap()

  await waitFor(() => expect(screen.getByText('Newsletter starter')).toBeInTheDocument())

  const user = userEvent.setup()
  // Click the first card
  await user.click(screen.getByRole('button', { name: 'Newsletter starter' }))

  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())

  // Select brand via @base-ui Select (trigger = combobox, options = option)
  const combobox = screen.getByRole('combobox')
  await user.click(combobox)
  await waitFor(() => expect(screen.getByRole('option', { name: 'Acme' })).toBeInTheDocument())
  await user.click(screen.getByRole('option', { name: 'Acme' }))

  // Type subject
  await user.type(screen.getByLabelText('Subject'), 'My Newsletter')

  // Submit
  await user.click(screen.getByRole('button', { name: 'Use template' }))

  await waitFor(() =>
    expect(capturedBody).toEqual({ brandId: 'b1', subject: 'My Newsletter' }),
  )
})
