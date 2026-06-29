import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { ListDetailPage } from '@/features/subscribers/ListDetailPage'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

const sampleList = {
  id: 'l1',
  owner_id: 'o',
  brand_id: 'b1',
  name: 'News',
  created_at: '',
}

const sampleSubscriber = {
  id: 's1',
  owner_id: 'o',
  list_id: 'l1',
  email: 'a@x.test',
  name: 'A',
  status: 'active',
  created_at: '',
}

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/lists/l1']}>
        <Routes>
          <Route path="/lists/:id" element={<ListDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

test('shows list heading and subscriber email', async () => {
  server.use(
    http.get('/api/lists', () => HttpResponse.json([sampleList])),
    http.get('/api/lists/l1/subscribers', () => HttpResponse.json([sampleSubscriber])),
  )

  wrap()

  await waitFor(() => expect(screen.getByText('News')).toBeInTheDocument())
  await waitFor(() => expect(screen.getByText('a@x.test')).toBeInTheDocument())
})

test('POST /api/lists/l1/subscribers receives correct body', async () => {
  let capturedBody: unknown

  server.use(
    http.get('/api/lists', () => HttpResponse.json([sampleList])),
    http.get('/api/lists/l1/subscribers', () => HttpResponse.json([])),
    http.post('/api/lists/l1/subscribers', async ({ request }) => {
      capturedBody = await request.json()
      return HttpResponse.json({
        id: 's2',
        owner_id: 'o',
        list_id: 'l1',
        email: 'b@x.test',
        name: 'B',
        status: 'active',
        created_at: '',
      })
    }),
  )

  wrap()

  await waitFor(() => expect(screen.getByText('No subscribers yet')).toBeInTheDocument())

  const user = userEvent.setup()
  await user.click(screen.getAllByRole('button', { name: 'Add subscriber' })[0])

  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())

  await user.type(screen.getByLabelText('Email'), 'b@x.test')
  await user.type(screen.getByLabelText('Name'), 'B')

  await user.click(screen.getByRole('button', { name: 'Add' }))

  await waitFor(() =>
    expect(capturedBody).toEqual({ email: 'b@x.test', name: 'B' }),
  )
})
