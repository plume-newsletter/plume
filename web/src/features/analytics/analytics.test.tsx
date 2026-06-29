import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { AnalyticsPage } from '@/features/analytics/AnalyticsPage'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

const overview = {
  subscribers: 100, netNewSubs: 12, avgOpenRate: 0.5, clickRate: 0.1, sendCost: 1.25,
  subscriberGrowth: [{ date: '2026-06-01', gained: 5, lost: 1 }],
  sendVolume: [], bestSendTimes: [{ label: 'Tue 9 AM', rate: 1 }],
  campaigns: [], topCampaigns: [{ id: 'c1', subject: 'Spring update', opens: 40 }],
}

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}><MemoryRouter><AnalyticsPage /></MemoryRouter></QueryClientProvider>,
  )
}

test('renders the analytics cards and a window toggle re-queries', async () => {
  let lastWindow = ''
  server.use(http.get('/api/analytics/overview', ({ request }) => {
    lastWindow = new URL(request.url).searchParams.get('window') ?? ''
    return HttpResponse.json(overview)
  }))
  wrap()
  await waitFor(() => expect(screen.getByText('Net new subs')).toBeInTheDocument())
  expect(screen.getByText('+12')).toBeInTheDocument()
  expect(screen.getByText('Spring update')).toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: /last 90 days/i }))
  await waitFor(() => expect(lastWindow).toBe('90'))
})
