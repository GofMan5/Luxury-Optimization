export interface TreemapValue {
  id: string
  size: number
}

export interface TreemapRect extends TreemapValue {
  x: number
  y: number
  width: number
  height: number
}

interface AreaValue extends TreemapValue { area: number }
interface Rect { x: number; y: number; width: number; height: number }

export function squarify(values: TreemapValue[], width = 100, height = 60): TreemapRect[] {
  const usable = values.filter((value) => value.size > 0 && Number.isFinite(value.size)).slice().sort((left, right) => right.size - left.size)
  const total = usable.reduce((sum, value) => sum + value.size, 0)
  if (total <= 0 || width <= 0 || height <= 0) return []
  const queue: AreaValue[] = usable.map((value) => ({ ...value, area: value.size / total * width * height }))
  const result: TreemapRect[] = []
  const remaining: Rect = { x: 0, y: 0, width, height }
  let row: AreaValue[] = []
  while (queue.length) {
    const next = queue[0]!
    const side = Math.min(remaining.width, remaining.height)
    if (row.length === 0 || worst([...row, next], side) <= worst(row, side)) {
      row.push(queue.shift()!)
    } else {
      layoutRow(row, remaining, result)
      row = []
    }
  }
  if (row.length) layoutRow(row, remaining, result)
  return result
}

function worst(row: AreaValue[], side: number): number {
  if (!row.length || side <= 0) return Number.POSITIVE_INFINITY
  const sum = row.reduce((value, item) => value + item.area, 0)
  const largest = Math.max(...row.map((item) => item.area))
  const smallest = Math.min(...row.map((item) => item.area))
  return Math.max(side * side * largest / (sum * sum), sum * sum / (side * side * smallest))
}

function layoutRow(row: AreaValue[], remaining: Rect, output: TreemapRect[]): void {
  const area = row.reduce((sum, value) => sum + value.area, 0)
  if (remaining.width >= remaining.height) {
    const rowWidth = remaining.height > 0 ? area / remaining.height : 0
    let y = remaining.y
    for (const value of row) {
      const itemHeight = rowWidth > 0 ? value.area / rowWidth : 0
      output.push({ id: value.id, size: value.size, x: remaining.x, y, width: rowWidth, height: itemHeight })
      y += itemHeight
    }
    remaining.x += rowWidth
    remaining.width = Math.max(0, remaining.width - rowWidth)
  } else {
    const rowHeight = remaining.width > 0 ? area / remaining.width : 0
    let x = remaining.x
    for (const value of row) {
      const itemWidth = rowHeight > 0 ? value.area / rowHeight : 0
      output.push({ id: value.id, size: value.size, x, y: remaining.y, width: itemWidth, height: rowHeight })
      x += itemWidth
    }
    remaining.y += rowHeight
    remaining.height = Math.max(0, remaining.height - rowHeight)
  }
}
