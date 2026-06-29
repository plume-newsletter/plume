import { render, screen } from '@testing-library/react'
import { expect, test } from 'vitest'
import { StatCard } from './StatCard'
import { EmptyState } from './EmptyState'
import { Inbox } from 'lucide-react'

test('StatCard shows label and value', () => {
  render(<StatCard label="Recipients" value={42} />)
  expect(screen.getByText('Recipients')).toBeInTheDocument()
  expect(screen.getByText('42')).toBeInTheDocument()
})

test('EmptyState shows title and action', () => {
  render(<EmptyState icon={Inbox} title="No lists yet" action={<button>New list</button>} />)
  expect(screen.getByText('No lists yet')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'New list' })).toBeInTheDocument()
})
