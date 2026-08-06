import { useCallback, useEffect, useState, type DependencyList } from 'react'

interface Resource<T> {
  data: T | null
  error: string | null
  loading: boolean
  refresh: () => void
}

export function useBackendResource<T>(load: (signal: AbortSignal) => Promise<T>, dependencies: DependencyList): Resource<T> {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [revision, setRevision] = useState(0)

  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    setError(null)
    load(controller.signal).then(setData).catch((reason: unknown) => {
      if (!controller.signal.aborted) setError(reason instanceof Error ? reason.message : String(reason))
    }).finally(() => {
      if (!controller.signal.aborted) setLoading(false)
    })
    return () => controller.abort()
  }, [...dependencies, revision]) // eslint-disable-line react-hooks/exhaustive-deps

  const refresh = useCallback(() => setRevision((value) => value + 1), [])
  return { data, error, loading, refresh }
}
