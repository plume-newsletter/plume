import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { SettingsPage } from '@/features/settings/SettingsPage'
import { ThemeProvider } from '@/components/theme/ThemeProvider'

beforeAll(() => server.listen()); afterEach(() => server.resetHandlers()); afterAll(() => server.close())

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}><ThemeProvider><MemoryRouter><SettingsPage /></MemoryRouter></ThemeProvider></QueryClientProvider>)
}

test('Team tab lists members and invites a member (owner)', async () => {
  let invited: unknown
  server.use(
    http.get('/api/me', () => HttpResponse.json({ email: 'o@x.test', fullName: 'Owner', role: 'owner', workspaceName: 'Acme' })),
    http.get('/api/settings', () => HttpResponse.json({ sesConfigured: false, aiConfigured: false })),
    http.get('/api/team', () => HttpResponse.json([{ id: 'u1', email: 'o@x.test', fullName: 'Owner', role: 'owner' }])),
    http.get('/api/team/invites', () => HttpResponse.json([])),
    http.post('/api/team/invites', async ({ request }) => { invited = await request.json(); return HttpResponse.json({ invite: { id: 'i1', email: 'new@x.test', role: 'editor', token: 't', expiresAt: '' }, acceptUrl: 'http://x/accept/t' }) }),
  )
  wrap()
  await userEvent.click(await screen.findByRole('button', { name: /^team$/i }))
  await waitFor(() => expect(screen.getByText('o@x.test')).toBeInTheDocument())
  await userEvent.click(screen.getByRole('button', { name: /invite member/i }))
  await userEvent.type(screen.getByLabelText(/email/i), 'new@x.test')
  await userEvent.click(screen.getByRole('button', { name: /send invite/i }))
  await waitFor(() => expect((invited as { email?: string })?.email).toBe('new@x.test'))
  await waitFor(() => expect(screen.getByText('http://x/accept/t')).toBeInTheDocument())
})
