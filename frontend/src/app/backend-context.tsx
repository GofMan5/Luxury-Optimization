import { createContext, useContext, useMemo, type PropsWithChildren } from 'react'
import { createBackendClient } from '../platform/backend/create-client'
import type { BackendClient } from '../platform/backend/client'

interface BackendContextValue {
  client: BackendClient
  preview: boolean
}

const BackendContext = createContext<BackendContextValue | null>(null)

export function BackendProvider({ children }: PropsWithChildren) {
  const value = useMemo(createBackendClient, [])
  return <BackendContext value={value}>{children}</BackendContext>
}

export function useBackend(): BackendContextValue {
  const value = useContext(BackendContext)
  if (!value) throw new Error('BackendProvider is missing.')
  return value
}
