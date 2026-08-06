import { describe, expect, it } from 'vitest'
import { parseBenchmarkText } from './benchmark-import'

describe('benchmark capture import', () => {
  it('normalizes CapFrameX JSON and MangoHud CSV into comparable runs', () => {
    const frames = Array.from({ length: 100 }, () => 10)
    expect(parseBenchmarkText('capframex.json', JSON.stringify({ Runs: [{ CaptureData: { MsBetweenPresents: frames } }] }))).toEqual([
      { average_fps: 100, one_percent_low_fps: 100, p95_frame_ms: 10 },
    ])

    const csv = ['os,cpu,gpu', 'Linux,CPU,GPU', 'fps,frametime,cpu_load', ...Array.from({ length: 100 }, () => '125,8,20')].join('\n')
    expect(parseBenchmarkText('mangohud.csv', csv)).toEqual([
      { average_fps: 125, one_percent_low_fps: 125, p95_frame_ms: 8 },
    ])
  })

  it('accepts the native Luxury JSON shape and rejects undersampled captures', () => {
    const run = { average_fps: 144, one_percent_low_fps: 100, p95_frame_ms: 8.5 }
    expect(parseBenchmarkText('luxury.json', JSON.stringify({ label: 'before', runs: [run, run, run] }))).toHaveLength(3)
    expect(() => parseBenchmarkText('short.csv', `fps,frametime\n${Array.from({ length: 5 }, () => '100,10').join('\n')}`)).toThrow('at least 30')
  })
})
