import { render, screen } from '@testing-library/react'
import { expect, test } from 'vitest'
import { ComingSoon } from '@/components/ComingSoon'

test('renders the feature title, teaser, and a Coming soon label', () => {
  render(<ComingSoon title="Segments" description="Saved filters to target slices of your audience." />)
  expect(screen.getByRole('heading', { name: 'Segments' })).toBeInTheDocument()
  expect(screen.getByText('Saved filters to target slices of your audience.')).toBeInTheDocument()
  expect(screen.getByText(/coming soon/i)).toBeInTheDocument()
})
