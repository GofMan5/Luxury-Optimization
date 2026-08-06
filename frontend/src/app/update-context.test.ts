import { describe, expect, it } from 'vitest'
import { isSupportedUpdateVersion } from './update-context'

describe('desktop updater release line', () => {
  it('accepts only stable 1.0.x versions', () => {
    expect(isSupportedUpdateVersion('1.0.1')).toBe(true)
    expect(isSupportedUpdateVersion('1.0.1000')).toBe(true)
    for (const version of ['v1.0.2', '1.1.0', '1.0.2-beta.1', '../1.0.2']) {
      expect(isSupportedUpdateVersion(version)).toBe(false)
    }
  })
})
