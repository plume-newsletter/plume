import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { BrandsPage } from '@/features/brands/BrandsPage'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

const sampleBrand = {
  id: 'b1',
  owner_id: 'o',
  name: 'Acme',
  from_name: 'Acme',
  from_email: 'n@acme.test',
  reply_to: '',
  created_at: '',
}

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/brands']}>
        <BrandsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

test('shows brand row after GET /api/brands', async () => {
  server.use(
    http.get('/api/brands', () => HttpResponse.json([sampleBrand])),
  )

  wrap()

  await waitFor(() => expect(screen.getAllByText('Acme')[0]).toBeInTheDocument())
})

test('POST /api/brands receives camelCase body', async () => {
  let capturedBody: unknown

  server.use(
    http.get('/api/brands', () => HttpResponse.json([])),
    http.post('/api/brands', async ({ request }) => {
      capturedBody = await request.json()
      return HttpResponse.json({
        id: 'b2',
        owner_id: 'o',
        name: 'Beta',
        from_name: 'Beta Sender',
        from_email: 'beta@beta.test',
        reply_to: 'reply@beta.test',
        created_at: '',
      })
    }),
  )

  wrap()

  // Wait for the empty state to appear, then open New brand dialog
  await waitFor(() => expect(screen.getByText('No brands yet')).toBeInTheDocument())

  const user = userEvent.setup()
  await user.click(screen.getAllByRole('button', { name: 'New brand' })[0])

  await waitFor(() => expect(screen.getByLabelText('Name')).toBeInTheDocument())

  await user.type(screen.getByLabelText('Name'), 'Beta')
  await user.type(screen.getByLabelText('From name'), 'Beta Sender')
  await user.type(screen.getByLabelText('From email'), 'beta@beta.test')
  await user.type(screen.getByLabelText('Reply-to'), 'reply@beta.test')

  await user.click(screen.getByRole('button', { name: 'Create' }))

  await waitFor(() =>
    expect(capturedBody).toEqual({
      name: 'Beta',
      fromName: 'Beta Sender',
      fromEmail: 'beta@beta.test',
      replyTo: 'reply@beta.test',
    }),
  )
})
