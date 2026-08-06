export interface BackendClient {
  call<T>(method: string, payload?: unknown, signal?: AbortSignal): Promise<T>
  invalidate(methodPrefix?: string): void
  stop(): Promise<void>
}
