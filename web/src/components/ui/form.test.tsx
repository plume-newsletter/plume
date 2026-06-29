import { render, screen } from '@testing-library/react'
import { useForm } from 'react-hook-form'
import {
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage,
} from '@/components/ui/form'

function TestForm() {
  const form = useForm({ defaultValues: { email: '' } })

  return (
    <Form {...form}>
      <FormField
        control={form.control}
        name="email"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Email</FormLabel>
            <FormControl>
              <input {...field} />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
    </Form>
  )
}

test('FormControl forwards a11y attrs onto its child input, not a wrapper div', () => {
  render(<TestForm />)

  const input = screen.getByLabelText('Email')
  expect(input.tagName).toBe('INPUT')
  expect(input).toHaveAttribute('id')
  expect(input.id).not.toBe('')
  expect(input).toHaveAttribute('aria-describedby')

  const label = screen.getByText('Email')
  expect(label).toHaveAttribute('for', input.id)
})
