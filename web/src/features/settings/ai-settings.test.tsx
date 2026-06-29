// src/web/src/features/settings/ai-settings.test.tsx
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
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
        <SettingsPage />
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

test('saves the Anthropic key via PUT /api/settings/ai', async () => {
  let received: unknown = null
  server.use(
    http.get('/api/settings', () =>
      HttpResponse.json({ sesConfigured: false, sesRegion: '', aiConfigured: false, aiModel: '' })),
    http.put('/api/settings/ai', async ({ request }) => {
      received = await request.json()
      return new HttpResponse(null, { status: 204 })
    }),
  )
  wrap()

  const key = await screen.findByLabelText('Anthropic API Key')
  await userEvent.type(key, 'sk-ant-xyz')
  await userEvent.click(screen.getByRole('button', { name: /save ai/i }))

  await waitFor(() => expect(received).toMatchObject({ apiKey: 'sk-ant-xyz' }))
})
