import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { AutomationsPage } from '@/features/automations/AutomationsPage'

beforeAll(() => server.listen()); afterEach(() => server.resetHandlers()); afterAll(() => server.close())

const auto = { id: 'a1', name: 'Welcome series', listId: 'l1', status: 'live', createdAt: '', stepSends: 1, inFlow: 2140, completePct: 0.64,
  steps: [{ kind: 'send', subject: 'Welcome 👋', html: '<p>hi</p>', waitDays: 0 }, { kind: 'wait', subject: '', html: '', waitDays: 2 }] }

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}><MemoryRouter><AutomationsPage /></MemoryRouter></QueryClientProvider>)
}

test('renders an automation card and its journey on select', async () => {
  server.use(
    http.get('/api/automations', () => HttpResponse.json([auto])),
    http.get('/api/lists', () => HttpResponse.json([{ id: 'l1', owner_id: 'o', brand_id: 'b1', name: 'Newsletter', created_at: '' }])),
  )
  wrap()
  await waitFor(() => expect(screen.getByText('Welcome series')).toBeInTheDocument())
  expect(screen.getByText(/1 email/i)).toBeInTheDocument()  // "1 emails · triggered on subscribe"
  // selecting the card shows the journey trigger + the send step subject
  await userEvent.click(screen.getByText('Welcome series'))
  await waitFor(() => expect(screen.getByText('Welcome 👋')).toBeInTheDocument())
  expect(screen.getByText(/subscriber joins/i)).toBeInTheDocument()
})
