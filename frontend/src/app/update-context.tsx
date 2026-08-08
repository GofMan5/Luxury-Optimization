import { createContext, useCallback, useContext, useEffect, useRef, useState, type PropsWithChildren } from 'react'
import { relaunch } from '@tauri-apps/plugin-process'
import { check as checkForUpdate, type Update } from '@tauri-apps/plugin-updater'
import { useBackend } from './backend-context'
import type { MutationResult, UpdateStatus } from '../shared/contracts/domain'

interface UpdateContextValue {
  status: UpdateStatus | null
  loading: boolean
  busy: 'check' | 'install' | null
  progress: number | null
  error: string | null
  checkNow: () => Promise<UpdateStatus>
  install: () => Promise<string>
}

const UpdateContext = createContext<UpdateContextValue | null>(null)
const releaseVersion = /^1\.0\.\d+$/
const automaticCheckDelayMs = 2_500

export function isSupportedUpdateVersion(value: string): boolean { return releaseVersion.test(value) }

export function UpdateProvider({ children }: PropsWithChildren) {
  const { client, preview } = useBackend()
  const [status, setStatus] = useState<UpdateStatus | null>(null)
  const [busy, setBusy] = useState<'check' | 'install' | null>(null)
  const [progress, setProgress] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)
  const update = useRef<Update | null>(null)
  const checking = useRef(false)
  const checked = useRef(false)

  const checkNow = useCallback(async (): Promise<UpdateStatus> => {
    if (checking.current) throw new Error('Update check is already running.')
    checking.current = true; checked.current = true; setBusy('check'); setError(null)
    try {
      const current = await client.call<UpdateStatus>('updates.status', {})
      if (preview) {
        const next = await client.call<UpdateStatus>('updates.check', {})
        setStatus(next)
        return next
      }
      const candidate = await checkForUpdate({ timeout: 20_000 })
      if (candidate && !isSupportedUpdateVersion(candidate.version)) {
        await candidate.close()
        throw new Error(`Unsupported update line: ${candidate.version}`)
      }
      if (update.current && update.current !== candidate) await update.current.close()
      update.current = candidate
      const next: UpdateStatus = {
        ...current,
        current: candidate?.currentVersion ?? current.current,
        latest: candidate?.version,
        update_ready: candidate !== null,
        last_check: new Date().toISOString(),
      }
      setStatus(next)
      return next
    } catch (reason) {
      const message = reason instanceof Error ? reason.message : String(reason)
      setError(message)
      throw reason
    } finally {
      checking.current = false; setBusy(null)
    }
  }, [client, preview])

  const install = useCallback(async (): Promise<string> => {
    setBusy('install'); setProgress(0); setError(null)
    try {
      if (preview) {
        const result = await client.call<MutationResult>('updates.install', {})
        return result.message
      }
      const candidate = update.current
      if (!candidate) throw new Error('Check for updates before installation.')
      let downloaded = 0
      let total = 0
      await candidate.downloadAndInstall((event) => {
        if (event.event === 'Started') total = event.data.contentLength ?? 0
        if (event.event === 'Progress') downloaded += event.data.chunkLength
        if (event.event === 'Finished') setProgress(100)
        else if (total > 0) setProgress(Math.min(99, Math.round(downloaded * 100 / total)))
      }, { timeout: 10 * 60_000 })
      await relaunch()
      return 'Update installed.'
    } catch (reason) {
      const message = reason instanceof Error ? reason.message : String(reason)
      setError(message)
      throw reason
    } finally {
      setBusy(null); setProgress(null)
    }
  }, [client, preview])

  useEffect(() => {
    // Let the first screen become interactive before updater network and signature work.
    const timer = window.setTimeout(() => { if (!checked.current) void checkNow().catch(() => undefined) }, automaticCheckDelayMs)
    return () => window.clearTimeout(timer)
  }, [checkNow])

  return <UpdateContext value={{ status, loading: status === null && error === null, busy, progress, error, checkNow, install }}>{children}</UpdateContext>
}

export function useUpdates(): UpdateContextValue {
  const value = useContext(UpdateContext)
  if (!value) throw new Error('UpdateProvider is missing.')
  return value
}
