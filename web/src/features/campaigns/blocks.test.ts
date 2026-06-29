import { expect, test } from 'vitest'
import { newBlock, addBlock, moveBlock, removeBlock, updateBlock, type Block } from './blocks'

test('newBlock creates a typed block with an id and sane defaults', () => {
  const b = newBlock('heading')
  expect(b.type).toBe('heading')
  expect(b.id).toBeTruthy()
  expect(b.level).toBe(2)
})

test('add/move/remove/update are immutable', () => {
  let list: Block[] = []
  list = addBlock(list, newBlock('text'))
  list = addBlock(list, newBlock('divider'))
  expect(list.map((b) => b.type)).toEqual(['text', 'divider'])

  const moved = moveBlock(list, 0, 1)
  expect(moved.map((b) => b.type)).toEqual(['divider', 'text'])
  expect(list.map((b) => b.type)).toEqual(['text', 'divider']) // original untouched

  const firstId = list[0].id
  const updated = updateBlock(list, firstId, { html: 'hi' })
  expect(updated[0].html).toBe('hi')
  expect(list[0].html).toBe('Write something…')

  const removed = removeBlock(list, firstId)
  expect(removed.map((b) => b.type)).toEqual(['divider'])
})
