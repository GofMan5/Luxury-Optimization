import { describe, expect, it } from 'vitest'
import { squarify } from './treemap'

describe('squarify', () => {
  it('preserves relative area and keeps rectangles inside the canvas', () => {
    const rectangles = squarify([{ id: 'a', size: 60 }, { id: 'b', size: 30 }, { id: 'c', size: 10 }], 100, 60)
    expect(rectangles).toHaveLength(3)
    const area = new Map(rectangles.map((item) => [item.id, item.width * item.height]))
    expect(area.get('a')).toBeCloseTo(3600, 6)
    expect(area.get('b')).toBeCloseTo(1800, 6)
    expect(area.get('c')).toBeCloseTo(600, 6)
    for (const item of rectangles) {
      expect(item.x).toBeGreaterThanOrEqual(0)
      expect(item.y).toBeGreaterThanOrEqual(0)
      expect(item.x + item.width).toBeLessThanOrEqual(100.000001)
      expect(item.y + item.height).toBeLessThanOrEqual(60.000001)
    }
  })

  it('drops invalid and zero-sized values', () => {
    expect(squarify([{ id: 'zero', size: 0 }, { id: 'bad', size: Number.NaN }])).toEqual([])
  })
})
