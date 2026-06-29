import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { CampaignsPage } from '@/features/campaigns/CampaignsPage'

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

const sampleCampaign = {
  id: 'c1',
  owner_id: 'o',
  brand_id: 'b1',
  subject: 'Hello',
  status: 'draft',
  html_body: '',
  plain_body: '',
  scheduled_at: null,
  created_at: '',
}

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/campaigns']}>
        <CampaignsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

test('shows campaign row after GET /api/campaigns', async () => {
  server.use(
    http.get('/api/brands', () => HttpResponse.json([sampleBrand])),
    http.get('/api/campaigns', () => HttpResponse.json([sampleCampaign])),
    http.get('/api/analytics/overview', () => HttpResponse.json({
      subscribers: 0, netNewSubs: 0, avgOpenRate: 0, clickRate: 0, sendCost: 0,
      subscriberGrowth: [], sendVolume: [], bestSendTimes: [],
      campaigns: [{ id: 'c1', subject: 'Hello', status: 'draft', sent: 1200, openRate: 0.5, clickRate: 0.1 }],
      topCampaigns: [],
    })),
  )

  wrap()

  await waitFor(() => expect(screen.getByText('Hello')).toBeInTheDocument())
  await waitFor(() => expect(screen.getByText('1,200')).toBeInTheDocument())
  await waitFor(() => expect(screen.getByText('draft')).toBeInTheDocument())
})

test('POST /api/campaigns receives camelCase body', async () => {
  let capturedBody: unknown

  server.use(
    http.get('/api/brands', () => HttpResponse.json([sampleBrand])),
    http.get('/api/campaigns', () => HttpResponse.json([])),
    http.post('/api/campaigns', async ({ request }) => {
      capturedBody = await request.json()
      return HttpResponse.json({
        id: 'c2',
        owner_id: 'o',
        brand_id: 'b1',
        subject: 'My Subject',
        status: 'draft',
        html_body: '',
        plain_body: '',
        scheduled_at: null,
        created_at: '',
      })
    }),
  )

  wrap()

  // Wait for empty state, then open dialog
  await waitFor(() => expect(screen.getByText('No campaigns yet')).toBeInTheDocument())

  const user = userEvent.setup()
  // Two "New campaign" buttons exist (PageHeader + EmptyState); click the first (PageHeader)
  await user.click(screen.getAllByRole('button', { name: 'New campaign' })[0])

  // Wait for dialog to open (brand select is present)
  await waitFor(() => expect(screen.getByLabelText('Brand')).toBeInTheDocument())

  // Wait for dialog to open (dialog role present)
  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())

  // @base-ui Select trigger renders as role="combobox"
  const selectTrigger = screen.getByRole('combobox')
  await user.click(selectTrigger)

  // @base-ui Select items render as role="option"
  await waitFor(() => expect(screen.getByRole('option', { name: 'Acme' })).toBeInTheDocument())
  await user.click(screen.getByRole('option', { name: 'Acme' }))

  // Type subject
  await user.type(screen.getByLabelText('Subject'), 'My Subject')

  // Submit
  await user.click(screen.getByRole('button', { name: 'Create' }))

  await waitFor(() =>
    expect(capturedBody).toEqual({
      brandId: 'b1',
      subject: 'My Subject',
      htmlBody: '',
      plainBody: '',
    }),
  )
})
