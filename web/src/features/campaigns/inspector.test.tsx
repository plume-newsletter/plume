import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeAll, afterAll, afterEach, expect, test, vi } from 'vitest'
import { useState } from 'react'
import { server, http, HttpResponse } from '@/test/msw'
import { BlockInspector } from '@/features/campaigns/BlockInspector'
import { newBlock } from '@/features/campaigns/blocks'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function wrap(ui: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

test('spacer height edit updates the block height', async () => {
  function Harness() {
    const [b, setB] = useState(newBlock('spacer'))
    return <BlockInspector block={b} onChange={(patch) => setB((prev) => ({ ...prev, ...patch }))} />
  }
  wrap(<Harness />)
  const input = screen.getByLabelText(/height/i) as HTMLInputElement
  await userEvent.clear(input)
  await userEvent.type(input, '40')
  expect(input.value).toBe('40')
})

test('text block shows copy-assist and applies the AI result', async () => {
  server.use(http.post('/api/ai/rewrite', () => HttpResponse.json({ text: 'shorter' })))
  const onChange = vi.fn()
  const b = { ...newBlock('text'), html: 'make this shorter' }
  wrap(<BlockInspector block={b} onChange={onChange} />)
  await userEvent.click(screen.getByRole('button', { name: /shorten/i }))
  await vi.waitFor(() => expect(onChange).toHaveBeenCalledWith({ html: 'shorter' }))
})
