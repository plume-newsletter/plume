import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { ListsPage } from '@/features/lists/ListsPage'

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

const sampleList = {
  id: 'l1',
  owner_id: 'o',
  brand_id: 'b1',
  name: 'News',
  created_at: '',
}

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/lists']}>
        <ListsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

test('shows list row with brand name resolved', async () => {
  server.use(
    http.get('/api/brands', () => HttpResponse.json([sampleBrand])),
    http.get('/api/lists', () => HttpResponse.json([sampleList])),
  )

  wrap()

  await waitFor(() => expect(screen.getByText('News')).toBeInTheDocument())
  await waitFor(() => expect(screen.getByText('Acme')).toBeInTheDocument())
})

test('Select trigger shows brand name not id after selection', async () => {
  server.use(
    http.get('/api/brands', () => HttpResponse.json([sampleBrand])),
    http.get('/api/lists', () => HttpResponse.json([])),
  )

  wrap()

  await waitFor(() => expect(screen.getByText('No lists yet')).toBeInTheDocument())

  const user = userEvent.setup()
  await user.click(screen.getAllByRole('button', { name: 'New list' })[0])

  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())

  const selectTrigger = screen.getByRole('combobox')
  await user.click(selectTrigger)

  await waitFor(() => expect(screen.getByRole('option', { name: 'Acme' })).toBeInTheDocument())
  await user.click(screen.getByRole('option', { name: 'Acme' }))

  // The trigger should now display the name "Acme", not the raw id "b1"
  await waitFor(() => {
    const trigger = screen.getByRole('combobox')
    expect(trigger.textContent).toContain('Acme')
    expect(trigger.textContent).not.toContain('b1')
  })
})

test('POST /api/lists receives correct body with brandId', async () => {
  let capturedBody: unknown

  server.use(
    http.get('/api/brands', () => HttpResponse.json([sampleBrand])),
    http.get('/api/lists', () => HttpResponse.json([])),
    http.post('/api/lists', async ({ request }) => {
      capturedBody = await request.json()
      return HttpResponse.json({ id: 'l2', owner_id: 'o', brand_id: 'b1', name: 'Newsletter', created_at: '' })
    }),
  )

  wrap()

  await waitFor(() => expect(screen.getByText('No lists yet')).toBeInTheDocument())

  const user = userEvent.setup()
  await user.click(screen.getAllByRole('button', { name: 'New list' })[0])

  // Wait for dialog to open — look for dialog title
  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())

  // @base-ui Select trigger renders as role="combobox"
  const selectTrigger = screen.getByRole('combobox')
  await user.click(selectTrigger)

  // @base-ui Select items render as role="option"
  await waitFor(() => expect(screen.getByRole('option', { name: 'Acme' })).toBeInTheDocument())
  await user.click(screen.getByRole('option', { name: 'Acme' }))

  // Type the list name
  await user.type(screen.getByLabelText('Name'), 'Newsletter')

  // Submit
  await user.click(screen.getByRole('button', { name: 'Create' }))

  await waitFor(() =>
    expect(capturedBody).toEqual({ brandId: 'b1', name: 'Newsletter' })
  )
})
