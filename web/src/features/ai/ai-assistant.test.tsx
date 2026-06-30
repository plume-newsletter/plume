import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { AiAssistantProvider, useAiAssistant } from '@/features/ai/AiAssistant'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function Trigger() {
  const ai = useAiAssistant()
  return <button onClick={ai.toggle}>open-ai</button>
}

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <AiAssistantProvider>
        <Trigger />
      </AiAssistantProvider>
    </QueryClientProvider>,
  )
}

test('opening the panel and sending a message shows the reply', async () => {
  server.use(
    http.post('/api/ai/chat', () => HttpResponse.json({ reply: 'Here is your draft.' })),
  )
  const user = userEvent.setup()
  wrap()

  await user.click(screen.getByText('open-ai'))
  expect(screen.getByText(/What do you need\?/i)).toBeInTheDocument()

  await user.type(screen.getByLabelText('Ask Plume AI'), 'write me an email')
  await user.click(screen.getByLabelText('Send'))

  expect(screen.getByText('write me an email')).toBeInTheDocument()
  await waitFor(() => expect(screen.getByText('Here is your draft.')).toBeInTheDocument())
})

test('clicking a Try chip sends that prompt', async () => {
  server.use(
    http.post('/api/ai/chat', () => HttpResponse.json({ reply: 'Draft ready.' })),
  )
  const user = userEvent.setup()
  wrap()

  await user.click(screen.getByText('open-ai'))
  await user.click(screen.getByText(/Write a re-engagement email/i))
  await waitFor(() => expect(screen.getByText('Draft ready.')).toBeInTheDocument())
})
