import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { AcceptInvitePage } from '@/features/auth/AcceptInvitePage'

beforeAll(() => server.listen()); afterEach(() => server.resetHandlers()); afterAll(() => server.close())

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}><MemoryRouter initialEntries={['/accept/tok123']}>
    <Routes><Route path="/accept/:token" element={<AcceptInvitePage />} /><Route path="/" element={<div>Home</div>} /></Routes>
  </MemoryRouter></QueryClientProvider>)
}

test('shows the invite and accepts it', async () => {
  let body: unknown
  server.use(
    http.get('/api/invites/tok123', () => HttpResponse.json({ email: 'new@x.test', workspaceName: 'Acme' })),
    http.post('/api/invites/tok123/accept', async ({ request }) => { body = await request.json(); return HttpResponse.json({ email: 'new@x.test' }) }),
  )
  wrap()
  await waitFor(() => expect(screen.getByText(/Acme/)).toBeInTheDocument())
  await userEvent.type(screen.getByLabelText(/full name/i), 'New Person')
  await userEvent.type(screen.getByLabelText(/password/i), 'pw123456')
  await userEvent.click(screen.getByRole('button', { name: /accept|join/i }))
  await waitFor(() => expect((body as { fullName?: string })?.fullName).toBe('New Person'))
})
