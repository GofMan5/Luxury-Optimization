import { describe, expect, it } from 'vitest'
import { decodeResultFrame } from './protocol'

describe('sidecar protocol', () => {
  it('accepts a valid result', () => {
    expect(decodeResultFrame('{"v":1,"id":"x","type":"result","ok":true,"payload":{}}').id).toBe('x')
  })

  it('rejects malformed envelopes', () => {
    expect(() => decodeResultFrame('{"v":2,"id":"x","type":"result","ok":true}')).toThrow('invalid frame')
  })
})
