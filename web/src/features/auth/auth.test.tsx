import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { ProtectedRoute } from '@/features/auth/ProtectedRoute'
import { LoginPage } from '@/features/auth/LoginPage'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route element={<ProtectedRoute />}>
            <Route path="/" element={<div>secret dashboard</div>} />
          </Route>
          <Route path="/login" element={<div>login screen</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function wrapLogin() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

test('unauthenticated user is redirected to login', async () => {
  server.use(http.get('/api/me', () => new HttpResponse('unauthorized', { status: 401 })))
  wrap()
  await waitFor(() => expect(screen.getByText('login screen')).toBeInTheDocument())
})

test('authenticated user sees protected content', async () => {
  server.use(http.get('/api/me', () => HttpResponse.json({ email: 'a@plume.test' })))
  wrap()
  await waitFor(() => expect(screen.getByText('secret dashboard')).toBeInTheDocument())
})

test('shows the branded login chrome (GitHub, tagline)', () => {
  wrapLogin()
  expect(screen.getByRole('button', { name: /continue with github/i })).toBeInTheDocument()
  expect(screen.getByText(/your data never leaves your server/i)).toBeInTheDocument()
  expect(screen.getByText(/welcome back/i)).toBeInTheDocument()
})
