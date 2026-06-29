import { expect, test } from 'vitest'
import { timeGreeting, displayName } from '@/features/dashboard/greeting'

test('timeGreeting buckets the day correctly', () => {
  expect(timeGreeting(0)).toBe('Good morning')
  expect(timeGreeting(11)).toBe('Good morning')
  expect(timeGreeting(12)).toBe('Good afternoon')
  expect(timeGreeting(17)).toBe('Good afternoon')
  expect(timeGreeting(18)).toBe('Good evening')
  expect(timeGreeting(23)).toBe('Good evening')
})

test('displayName uses the email local part, falls back to "there"', () => {
  expect(displayName('admin@itthirit.io')).toBe('Admin')
  expect(displayName(undefined)).toBe('there')
})
