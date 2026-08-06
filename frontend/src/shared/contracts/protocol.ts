export const PROTOCOL_VERSION = 1 as const

export interface CommandFrame {
  v: typeof PROTOCOL_VERSION
  id: string
  type: 'command'
  method: string
  payload?: unknown
}

export interface ResultFrame {
  v: typeof PROTOCOL_VERSION
  id: string
  type: 'result'
  ok: boolean
  payload?: unknown
  error?: { code: string; message: string }
}

export class BackendError extends Error {
  constructor(readonly code: string, message: string) {
    super(message)
    this.name = 'BackendError'
  }
}

export function decodeResultFrame(text: string): ResultFrame {
  const frame: unknown = JSON.parse(text)
  if (!isRecord(frame) || frame.v !== PROTOCOL_VERSION || frame.type !== 'result' || typeof frame.id !== 'string' || typeof frame.ok !== 'boolean') {
    throw new BackendError('protocol_error', 'Backend returned an invalid frame.')
  }
  if (!frame.ok && (!isRecord(frame.error) || typeof frame.error.code !== 'string' || typeof frame.error.message !== 'string')) {
    throw new BackendError('protocol_error', 'Backend returned an invalid error frame.')
  }
  return frame as unknown as ResultFrame
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
