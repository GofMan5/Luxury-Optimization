import type { BackendClient } from './client'
import { CachedBackendClient } from './cached-client'
import { PreviewBackendClient } from './preview-client'
import { TauriBackendClient } from './tauri-client'

export function createBackendClient(): { client: BackendClient; preview: boolean } {
  const preview = window.__TAURI_INTERNALS__ === undefined
  const backend = preview ? new PreviewBackendClient() : new TauriBackendClient()
  return { client: new CachedBackendClient(backend), preview }
}
