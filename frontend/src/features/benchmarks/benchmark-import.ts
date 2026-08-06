import type { BenchmarkRun } from '../../shared/contracts/domain'

const maxFiles = 100
const maxFileBytes = 16 << 20
const maxTotalBytes = 64 << 20
const minFrameSamples = 30
const maxFrameSamples = 2_000_000
const maxMetric = 100_000

export async function importBenchmarkFiles(files: readonly File[]): Promise<BenchmarkRun[]> {
  if (files.length === 0 || files.length > maxFiles) throw new Error('Choose between 1 and 100 capture files.')
  if (files.reduce((total, file) => total + file.size, 0) > maxTotalBytes) throw new Error('Selected captures exceed the 64 MiB import limit.')
  const runs: BenchmarkRun[] = []
  for (const file of files) {
    if (file.size === 0 || file.size > maxFileBytes) throw new Error(`${file.name}: file must be between 1 byte and 16 MiB.`)
    runs.push(...parseBenchmarkText(file.name, await file.text()))
    if (runs.length > maxFiles) throw new Error('Imported captures contain more than 100 runs.')
  }
  if (runs.length < 3) throw new Error('Import at least three identical runs for a reliable comparison.')
  return runs
}

export function parseBenchmarkText(name: string, text: string): BenchmarkRun[] {
  const trimmed = text.trim()
  if (!trimmed) throw new Error(`${name}: capture is empty.`)
  try {
    const runs = trimmed.startsWith('{') || trimmed.startsWith('[') ? parseJSON(trimmed) : parseMangoHudCSV(trimmed)
    if (runs.length === 0) throw new Error('no supported runs found')
    return runs
  } catch (reason) {
    const message = reason instanceof Error ? reason.message : String(reason)
    throw new Error(`${name}: ${message}`)
  }
}

function parseJSON(text: string): BenchmarkRun[] {
  const root: unknown = JSON.parse(text)
  const record = asRecord(root)
  const candidates = Array.isArray(root) ? root : Array.isArray(record?.runs) ? record.runs : Array.isArray(record?.Runs) ? record.Runs : [root]
  return candidates.flatMap(runFromJSON)
}

function runFromJSON(value: unknown): BenchmarkRun[] {
  const record = asRecord(value)
  if (!record) return []
  const direct = {
    average_fps: finiteNumber(record.average_fps),
    one_percent_low_fps: finiteNumber(record.one_percent_low_fps),
    p95_frame_ms: finiteNumber(record.p95_frame_ms),
  }
  if (validRun(direct)) return [direct]
  const capture = asRecord(record.CaptureData) ?? asRecord(record.capture_data)
  const frames = numberArray(capture?.MsBetweenPresents ?? capture?.ms_between_presents)
  return frames.length >= minFrameSamples ? [runFromFrameTimes(frames)] : []
}

function parseMangoHudCSV(text: string): BenchmarkRun[] {
  const lines = text.split(/\r?\n/)
  for (let row = 0; row < lines.length; row++) {
    const line = lines[row]
    if (!line) continue
    for (const delimiter of [',', ';', '\t']) {
      const header = splitDelimited(line, delimiter).map(normalizeHeader)
      const fpsIndex = header.indexOf('fps')
      const frameIndex = header.findIndex((value) => value === 'frametime' || value === 'frametimems')
      if (fpsIndex < 0 || frameIndex < 0) continue
      const frames: number[] = []
      for (let dataRow = row + 1; dataRow < lines.length; dataRow++) {
        const dataLine = lines[dataRow]
        if (dataLine === undefined) continue
        const cells = splitDelimited(dataLine, delimiter)
        const fps = decimal(cells[fpsIndex], delimiter)
        const frame = decimal(cells[frameIndex], delimiter)
        if (fps > 0 && frame > 0 && Number.isFinite(fps) && Number.isFinite(frame)) frames.push(frame)
        if (frames.length > maxFrameSamples) throw new Error(`MangoHud capture exceeds ${maxFrameSamples} frame samples`)
      }
      if (frames.length < minFrameSamples) throw new Error(`MangoHud capture needs at least ${minFrameSamples} valid frame samples`)
      return [runFromFrameTimes(frames)]
    }
  }
  throw new Error('expected Luxury JSON, CapFrameX JSON, or MangoHud CSV')
}

function runFromFrameTimes(frames: number[]): BenchmarkRun {
  const ordered = frames.filter((value) => value > 0 && Number.isFinite(value)).sort((left, right) => left - right)
  if (ordered.length < minFrameSamples) throw new Error(`capture needs at least ${minFrameSamples} valid frame samples`)
  const averageFrame = ordered.reduce((total, value) => total + value, 0) / ordered.length
  return {
    average_fps: rounded(1000 / averageFrame),
    one_percent_low_fps: rounded(1000 / percentile(ordered, 0.99)),
    p95_frame_ms: rounded(percentile(ordered, 0.95)),
  }
}

function percentile(sorted: number[], fraction: number): number {
  const position = (sorted.length - 1) * fraction
  const lower = Math.floor(position)
  const upper = Math.ceil(position)
  const low = sorted[lower]
  const high = sorted[upper]
  if (low === undefined || high === undefined) throw new Error('capture has no frame samples')
  return low + (high - low) * (position - lower)
}

function splitDelimited(line: string, delimiter: string): string[] {
  const cells: string[] = []
  let cell = ''
  let quoted = false
  for (let index = 0; index < line.length; index++) {
    const value = line[index]
    if (value === '"') {
      if (quoted && line[index + 1] === '"') { cell += '"'; index++ } else quoted = !quoted
    } else if (value === delimiter && !quoted) {
      cells.push(cell); cell = ''
    } else cell += value
  }
  cells.push(cell)
  return cells
}

function normalizeHeader(value: string): string {
  return value.replace(/^\uFEFF/, '').trim().toLowerCase().replace(/[^a-z0-9]/g, '')
}

function decimal(value: string | undefined, delimiter: string): number {
  if (value === undefined) return Number.NaN
  const normalized = delimiter === ',' ? value.trim() : value.trim().replace(',', '.')
  return Number(normalized)
}

function numberArray(value: unknown): number[] {
  if (!Array.isArray(value)) return []
  if (value.length > maxFrameSamples) throw new Error(`capture exceeds ${maxFrameSamples} frame samples`)
  return value.map(finiteNumber).filter((item) => Number.isFinite(item))
}

function finiteNumber(value: unknown): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : Number.NaN
}

function validRun(run: BenchmarkRun): boolean {
  return Object.values(run).every((value) => value > 0 && value <= maxMetric && Number.isFinite(value))
}

function rounded(value: number): number {
  return Math.round(value * 1000) / 1000
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return typeof value === 'object' && value !== null && !Array.isArray(value) ? value as Record<string, unknown> : null
}
