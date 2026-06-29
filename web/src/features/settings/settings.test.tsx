import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { SettingsPage } from '@/features/settings/SettingsPage'
import { ThemeProvider } from '@/components/theme/ThemeProvider'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <ThemeProvider>
        <MemoryRouter initialEntries={['/settings']}>
          <SettingsPage />
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

test('when sesConfigured=true: shows connected panel, hides form, shows Replace button that reveals form', async () => {
  server.use(
    http.get('/api/settings', () =>
      HttpResponse.json({ sesConfigured: true, sesRegion: 'us-east-1' }),
    ),
  )

  const { queryByLabelText } = wrap()

  // Connected panel must appear
  await waitFor(() => expect(screen.getByText(/SES is connected/i)).toBeInTheDocument())

  // Region should be shown somewhere in the connected panel (badge and/or panel text)
  await waitFor(() => expect(screen.getAllByText(/us-east-1/).length).toBeGreaterThan(0))

  // Credential inputs must NOT be visible initially
  expect(queryByLabelText('AWS Secret Access Key')).toBeNull()
  expect(queryByLabelText('AWS Access Key ID')).toBeNull()

  // Replace credentials button must be present
  const replaceBtn = screen.getByRole('button', { name: /Replace credentials/i })
  expect(replaceBtn).toBeInTheDocument()

  // Clicking Replace reveals the form
  const user = userEvent.setup()
  await user.click(replaceBtn)

  await waitFor(() =>
    expect(screen.getByLabelText('AWS Secret Access Key')).toBeInTheDocument(),
  )
  expect(screen.getByLabelText('AWS Access Key ID')).toBeInTheDocument()

  // Cancel button hides the form again
  const cancelBtn = screen.getByRole('button', { name: /Cancel/i })
  await user.click(cancelBtn)

  await waitFor(() =>
    expect(queryByLabelText('AWS Secret Access Key')).toBeNull(),
  )
})

test('when sesConfigured=false: shows credential form directly', async () => {
  server.use(
    http.get('/api/settings', () =>
      HttpResponse.json({ sesConfigured: false, sesRegion: '' }),
    ),
  )

  wrap()

  await waitFor(() =>
    expect(screen.getByLabelText('AWS Access Key ID')).toBeInTheDocument(),
  )
  expect(screen.getByLabelText('AWS Secret Access Key')).toBeInTheDocument()
  expect(screen.getByLabelText('Region')).toBeInTheDocument()
})

test('shows not-configured badge, saves SES credentials, then shows connected badge after refetch', async () => {
  let getCalls = 0
  server.use(
    http.get('/api/settings', () => {
      getCalls += 1
      return getCalls === 1
        ? HttpResponse.json({ sesConfigured: false, sesRegion: '' })
        : HttpResponse.json({ sesConfigured: true, sesRegion: 'us-east-1' })
    }),
  )

  let capturedBody: unknown
  server.use(
    http.put('/api/settings/ses', async ({ request }) => {
      capturedBody = await request.json()
      return new HttpResponse(null, { status: 204 })
    }),
  )

  wrap()

  await waitFor(() => expect(screen.getByText('SES not configured')).toBeInTheDocument())

  const user = userEvent.setup()
  await user.type(screen.getByLabelText('AWS Access Key ID'), 'AKIAEXAMPLE')
  await user.type(screen.getByLabelText('AWS Secret Access Key'), 'super-secret-key')
  await user.type(screen.getByLabelText('Region'), 'us-east-1')
  await user.click(screen.getByRole('button', { name: 'Save SES credentials' }))

  await waitFor(() =>
    expect(capturedBody).toEqual({
      accessKeyId: 'AKIAEXAMPLE',
      secretAccessKey: 'super-secret-key',
      region: 'us-east-1',
    }),
  )

  await waitFor(() => expect(screen.getByText('SES connected (us-east-1)')).toBeInTheDocument())
  expect(getCalls).toBe(2)
})
