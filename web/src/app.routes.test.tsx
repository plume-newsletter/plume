import { expect, test } from 'vitest'
import { router } from '@/App'

function allPaths(routes: typeof router.routes): string[] {
  const out: string[] = []
  for (const r of routes) {
    if (r.path) out.push(r.path)
    if (r.children) out.push(...allPaths(r.children as typeof router.routes))
  }
  return out
}

test('registers a route for every placeholder feature', () => {
  const paths = allPaths(router.routes)
  for (const p of ['analytics', 'segments', 'signup-forms', 'automations', 'ab-tests', 'deliverability', 'ai']) {
    expect(paths).toContain(p)
  }
})
