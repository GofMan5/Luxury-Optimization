import { invoke } from '@tauri-apps/api/core'
import { listen, type UnlistenFn } from '@tauri-apps/api/event'
import { BackendError, decodeResultFrame, PROTOCOL_VERSION, type CommandFrame } from '../../shared/contracts/protocol'
import type { BackendClient } from './client'

interface PendingCall {
  resolve: (payload: unknown) => void
  reject: (error: Error) => void
  timeout: number
}

export class TauriBackendClient implements BackendClient {
  #started = false
  #starting: Promise<void> | null = null
  #pending = new Map<string, PendingCall>()
  #unlisten: UnlistenFn[] = []

  async call<T>(method: string, payload?: unknown, signal?: AbortSignal): Promise<T> {
    if (signal?.aborted) throw new DOMException('Backend request aborted.', 'AbortError')
    await this.#start()
    return this.#request<T>(method, payload, signal)
  }

  async stop(): Promise<void> {
    if (this.#starting) {
      try { await this.#starting } catch { /* cleanup below */ }
    }
    if (this.#started) {
      try { await this.#request('system.shutdown', {}, undefined, 2_000) } catch { /* hard-stop below */ }
    }
    this.#disconnect(new BackendError('disconnected', 'Backend stopped.'))
    try { await invoke('sidecar_stop') } catch { /* process already exited */ }
    for (const unlisten of this.#unlisten.splice(0)) unlisten()
  }

  invalidate(): void {}

  #start(): Promise<void> {
    if (this.#started) return Promise.resolve()
    if (this.#starting) return this.#starting
    this.#starting = this.#spawn().finally(() => { this.#starting = null })
    return this.#starting
  }

  async #spawn(): Promise<void> {
    if (this.#unlisten.length === 0) {
      const unlistenFrame = await listen<string>('sidecar-frame', (event) => this.#consume(event.payload))
      try {
        const unlistenLifecycle = await listen('sidecar-lifecycle', () => this.#disconnect(new BackendError('disconnected', 'Backend connection closed.')))
        this.#unlisten.push(unlistenFrame, unlistenLifecycle)
      } catch (error) {
        unlistenFrame()
        throw error
      }
    }
    try {
      await invoke('sidecar_start')
      this.#started = true
      await this.#request('system.handshake', {}, undefined, 5_000)
    } catch (error) {
      this.#disconnect(asError(error))
      try { await invoke('sidecar_stop') } catch { /* preserve original error */ }
      throw error
    }
  }

  async #request<T>(method: string, payload?: unknown, signal?: AbortSignal, timeoutMS = commandTimeout(method)): Promise<T> {
    if (!this.#started) throw new BackendError('not_connected', 'Backend is not connected.')
    const id = requestID()
    const frame: CommandFrame = { v: PROTOCOL_VERSION, id, type: 'command', method, ...(payload === undefined ? {} : { payload }) }
    let abort = () => undefined
    const result = new Promise<T>((resolve, reject) => {
      const timeout = window.setTimeout(() => {
        this.#pending.delete(id)
        void this.#cancel(id)
        reject(new BackendError('timeout', `${method} timed out.`))
      }, timeoutMS)
      this.#pending.set(id, { resolve: (value) => resolve(value as T), reject, timeout })
      abort = () => {
        const pending = this.#pending.get(id)
        if (!pending) return
        this.#pending.delete(id)
        window.clearTimeout(pending.timeout)
        void this.#cancel(id)
        pending.reject(new DOMException('Backend request aborted.', 'AbortError'))
      }
    })
    try {
      await invoke('sidecar_write', { frame: JSON.stringify(frame) })
      if (signal?.aborted) abort()
      else signal?.addEventListener('abort', abort, { once: true })
      return await result
    } catch (error) {
      const pending = this.#pending.get(id)
      if (pending) {
        this.#pending.delete(id)
        window.clearTimeout(pending.timeout)
        pending.reject(asError(error))
      }
      throw error
    } finally {
      signal?.removeEventListener('abort', abort)
    }
  }

  async #cancel(id: string): Promise<void> {
    if (!this.#started) return
    const frame: CommandFrame = { v: PROTOCOL_VERSION, id: requestID(), type: 'command', method: 'system.cancel', payload: { id } }
    try { await invoke('sidecar_write', { frame: JSON.stringify(frame) }) } catch { /* original request owns the error */ }
  }

  #consume(text: string): void {
    let frame
    try { frame = decodeResultFrame(text) } catch (error) {
      this.#disconnect(asError(error))
      return
    }
    const pending = this.#pending.get(frame.id)
    if (!pending) return
    this.#pending.delete(frame.id)
    window.clearTimeout(pending.timeout)
    if (frame.ok) pending.resolve(frame.payload)
    else pending.reject(new BackendError(frame.error?.code ?? 'operation_failed', frame.error?.message ?? 'Operation failed.'))
  }

  #disconnect(error: Error): void {
    this.#started = false
    for (const pending of this.#pending.values()) {
      window.clearTimeout(pending.timeout)
      pending.reject(error)
    }
    this.#pending.clear()
  }
}

function requestID(): string {
  return crypto.randomUUID()
}

function commandTimeout(method: string): number {
  if (method === 'gaming.scan') return 2 * 60_000
  if (method === 'updates.install' || method === 'optimization.apply' || method === 'optimization.restore' || method === 'optimization.apply_tweak' || method === 'optimization.restore_tweak' || method === 'optimization.create_checkpoint' || method === 'services.set') return 10 * 60_000
  if (method === 'network.test') return 6 * 60_000
  return 30_000
}

function asError(value: unknown): Error {
  return value instanceof Error ? value : new Error(String(value))
}
