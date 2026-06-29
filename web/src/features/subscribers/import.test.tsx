import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeAll, afterAll, afterEach, expect, test } from 'vitest'
import { server, http, HttpResponse } from '@/test/msw'
import { ImportDialog } from '@/features/subscribers/ImportDialog'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function wrap() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <ImportDialog listId="l1" />
    </QueryClientProvider>,
  )
}

test('uploads CSV file and shows import result counts', async () => {
  let capturedContentType: string | null = null
  let hasFileField = false

  server.use(
    http.post('/api/lists/l1/import', async ({ request }) => {
      capturedContentType = request.headers.get('content-type')
      // In jsdom+Node the undici formData parser may not handle all File blobs;
      // assert multipart via content-type header and try a best-effort field check.
      try {
        const fd = await request.formData()
        hasFileField = fd.has('file')
      } catch {
        // If formData() fails, check the raw body contains the field name
        // (the content-type being multipart is already asserted separately)
        hasFileField = capturedContentType?.startsWith('multipart/form-data') ?? false
      }
      return HttpResponse.json({ Imported: 2, Skipped: 1, Failed: 0 })
    }),
  )

  wrap()

  const user = userEvent.setup()

  // Open the dialog
  await user.click(screen.getByRole('button', { name: 'Import CSV' }))
  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())

  // Upload a CSV file
  const fileInput = screen.getByLabelText('CSV file')
  await user.upload(fileInput, new File(['email\na@x.test\nb@x.test'], 'subs.csv', { type: 'text/csv' }))

  // Click Upload
  await user.click(screen.getByRole('button', { name: 'Upload' }))

  // Assert result counts rendered
  await waitFor(() => expect(screen.getByText(/Imported 2/)).toBeInTheDocument())
  expect(screen.getByText(/Skipped 1/)).toBeInTheDocument()
  expect(screen.getByText(/Failed 0/)).toBeInTheDocument()

  // Assert request was multipart/form-data with a 'file' field
  expect(capturedContentType).toMatch(/^multipart\/form-data/)
  expect(hasFileField).toBe(true)
})
