import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { SendDialog } from '@/features/campaigns/SendDialog'

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

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <SendDialog campaignId="c1" disabled={false} />
    </QueryClientProvider>,
  )
}

test('Test A: open dialog, select list, send; asserts body {listId} and success message', async () => {
  let capturedBody: unknown

  server.use(
    http.get('/api/lists', () => HttpResponse.json([sampleList])),
    http.post('/api/campaigns/c1/send', async ({ request }) => {
      capturedBody = await request.json()
      return HttpResponse.json({ recipients: 5 })
    }),
  )

  wrap()

  const user = userEvent.setup()

  // Open the dialog via the Send trigger button
  await user.click(screen.getByRole('button', { name: 'Send' }))

  // Wait for dialog to appear
  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())

  // Select the list — @base-ui Select trigger renders as role="combobox"
  const selectTrigger = screen.getByRole('combobox')
  await user.click(selectTrigger)

  // Wait for list option and click it
  await waitFor(() => expect(screen.getByRole('option', { name: 'News' })).toBeInTheDocument())
  await user.click(screen.getByRole('option', { name: 'News' }))

  // Click the Send button inside the dialog footer
  const sendButtons = screen.getAllByRole('button', { name: 'Send' })
  // The footer Send button is the last one (trigger is first)
  await user.click(sendButtons[sendButtons.length - 1])

  // Assert success message
  await waitFor(() =>
    expect(screen.getByText('Queued 5 recipients.')).toBeInTheDocument(),
  )

  // Assert the POST body had camelCase listId
  expect(capturedBody).toEqual({ listId: 'l1' })
})

test('Test B: 409 response maps to friendly error message', async () => {
  server.use(
    http.get('/api/lists', () => HttpResponse.json([sampleList])),
    http.post('/api/campaigns/c1/send', () =>
      HttpResponse.json({ error: 'already sent' }, { status: 409 }),
    ),
  )

  wrap()

  const user = userEvent.setup()

  // Open the dialog
  await user.click(screen.getByRole('button', { name: 'Send' }))

  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())

  // Select the list
  const selectTrigger = screen.getByRole('combobox')
  await user.click(selectTrigger)

  await waitFor(() => expect(screen.getByRole('option', { name: 'News' })).toBeInTheDocument())
  await user.click(screen.getByRole('option', { name: 'News' }))

  // Click Send in footer
  const sendButtons = screen.getAllByRole('button', { name: 'Send' })
  await user.click(sendButtons[sendButtons.length - 1])

  // Assert 409 error message
  await waitFor(() =>
    expect(
      screen.getByText('This campaign has already been sent or queued.'),
    ).toBeInTheDocument(),
  )
})
