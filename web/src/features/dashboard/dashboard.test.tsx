import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { DashboardPage } from '@/features/dashboard/DashboardPage'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

const sampleCampaigns = [
  { id: 'c1', owner_id: 'o', brand_id: 'b1', subject: 'Spring Sale', status: 'sent', html_body: '', plain_body: '', scheduled_at: null, created_at: '2024-02-01T00:00:00Z' },
  { id: 'c2', owner_id: 'o', brand_id: 'b1', subject: 'Welcome Email', status: 'draft', html_body: '', plain_body: '', scheduled_at: null, created_at: '2024-01-01T00:00:00Z' },
]

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <DashboardPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

test('renders greeting, prototype metric cards, and recent campaigns from real data', async () => {
  server.use(
    http.get('/api/me', () => HttpResponse.json({ email: 'admin@plume.test' })),
    http.get('/api/campaigns', () => HttpResponse.json(sampleCampaigns)),
    http.get('/api/analytics/overview', () => HttpResponse.json({
      subscribers: 1234, netNewSubs: 0, avgOpenRate: 0.421, clickRate: 0.073, sendCost: 7.77,
      subscriberGrowth: [], sendVolume: [], bestSendTimes: [],
      campaigns: [{ id: 'c1', subject: 'Spring Sale', status: 'sent', sent: 12480, openRate: 0.582, clickRate: 0.094 }],
      topCampaigns: [],
    })),
  )

  wrap()

  await waitFor(() =>
    expect(screen.getByText(/Good (morning|afternoon|evening), Admin 👋/)).toBeInTheDocument(),
  )

  // Prototype metric cards ('Click rate' also appears as a table column header)
  for (const label of ['Subscribers', 'Avg. open rate', 'Click rate', 'Spend (mo.)']) {
    expect(screen.getAllByText(label).length).toBeGreaterThan(0)
  }

  // Real analytics data renders
  await waitFor(() => expect(screen.getByText('1,234')).toBeInTheDocument())
  expect(screen.getByText('$7.77')).toBeInTheDocument()

  // Recent campaigns table is driven by analytics campaigns data
  await waitFor(() => expect(screen.getByText('Spring Sale')).toBeInTheDocument())
})

test('shows empty-state when there are no campaigns in analytics', async () => {
  server.use(
    http.get('/api/me', () => HttpResponse.json({ email: 'admin@plume.test' })),
    http.get('/api/campaigns', () => HttpResponse.json([])),
    http.get('/api/analytics/overview', () => HttpResponse.json({
      subscribers: 0, netNewSubs: 0, avgOpenRate: 0, clickRate: 0, sendCost: 0,
      subscriberGrowth: [], sendVolume: [], bestSendTimes: [],
      campaigns: [],
      topCampaigns: [],
    })),
  )

  wrap()

  await waitFor(() => expect(screen.getByText('No campaigns yet.')).toBeInTheDocument())
})
