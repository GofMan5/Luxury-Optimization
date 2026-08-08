import { describe, expect, it } from 'vitest'
import { dialogFocusTarget } from './dialog-focus'

describe('confirm dialog focus loop', () => {
  it('wraps only at the modal boundaries', () => {
    expect(dialogFocusTarget(0, 2, true)).toBe(2)
    expect(dialogFocusTarget(2, 2, false)).toBe(0)
    expect(dialogFocusTarget(1, 2, false)).toBeNull()
  })
})
