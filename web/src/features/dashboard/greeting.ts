export function timeGreeting(hour: number): string {
  if (hour < 12) return 'Good morning'
  if (hour < 18) return 'Good afternoon'
  return 'Good evening'
}

export function displayName(email: string | undefined): string {
  if (!email) return 'there'
  const local = email.split('@')[0]
  return local.charAt(0).toUpperCase() + local.slice(1)
}
