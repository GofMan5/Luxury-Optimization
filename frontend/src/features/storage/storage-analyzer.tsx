import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { listen, type UnlistenFn } from '@tauri-apps/api/event'
import { getCurrentWindow } from '@tauri-apps/api/window'
import { ArrowUp, File, Folder, HardDrive, RotateCw, Search, Trash2, X } from 'lucide-react'
import { useBackend } from '../../app/backend-context'
import { useLanguage } from '../../app/language-context'
import type { StorageDeletePreview, StorageDeleteResult, StorageScanExtension, StorageScanFile, StorageScanNode, StorageScanStart, StorageScanStatus } from '../../shared/contracts/domain'
import { Button } from '../../shared/ui/button'
import { ConfirmDialog } from '../../shared/ui/confirm-dialog'
import { InlineAlert } from '../../shared/ui/feedback'
import { squarify } from './treemap'

type AnalyzerView = 'folders' | 'files' | 'extensions'
type StartPayload = { path: string } | { parent_scan_id: string; node_id: string } | { refresh_scan_id: string }
type DeleteCandidate = { id: string; name: string; kind: 'directory' | 'file'; size: number; files: number; directories: number }

export function StorageAnalyzerWindow({ initialPath }: { initialPath: string }) {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = analyzerCopy[language]
  const request = useRef<AbortController | null>(null)
  const scanID = useRef<string | null>(null)
  const [scanRequest, setScanRequest] = useState({ path: initialPath, sequence: 0 })
  const volumePath = scanRequest.path
  const [status, setStatus] = useState<StorageScanStatus | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [mutationError, setMutationError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [view, setView] = useState<AnalyzerView>('folders')
  const [query, setQuery] = useState('')
  const [deletePreview, setDeletePreview] = useState<StorageDeletePreview | null>(null)
  const [deleteBusy, setDeleteBusy] = useState(false)
  const [typedName, setTypedName] = useState('')

  const cancel = useCallback(async () => {
    request.current?.abort()
    request.current = null
    const id = scanID.current
    if (id) {
      try { await client.call('storage.scan.cancel', { scan_id: id }) } catch { /* process exit or completed scan */ }
    }
  }, [client])

  const begin = useCallback(async (payload: StartPayload) => {
    await cancel()
    const controller = new AbortController()
    request.current = controller
    scanID.current = null
    setStatus(null); setError(null); setMutationError(null); setQuery(''); setView('folders'); setDeletePreview(null); setTypedName('')
    try {
      const started = await client.call<StorageScanStart>('storage.scan.start', payload, controller.signal)
      if (controller.signal.aborted) return
      scanID.current = started.scan_id
      setStatus({ scan_id: started.scan_id, state: 'scanning', root: started.root, started_at: started.started_at, elapsed_ms: 0, files_scanned: 0, directories_scanned: 0, bytes_scanned: 0, skipped: 0, cached: started.cached })
      if (started.cached) {
        setStatus(await client.call<StorageScanStatus>('storage.scan.status', { scan_id: started.scan_id }, controller.signal))
        return
      }
      while (!controller.signal.aborted) {
        await pause(250, controller.signal)
        const next = await client.call<StorageScanStatus>('storage.scan.status', { scan_id: started.scan_id }, controller.signal)
        if (controller.signal.aborted) return
        setStatus(next)
        if (next.state !== 'scanning') break
      }
    } catch (reason) {
      if (!controller.signal.aborted) setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      if (request.current === controller) request.current = null
    }
  }, [cancel, client])

  useEffect(() => {
    void begin({ path: volumePath })
    return () => { void cancel() }
  }, [begin, cancel, scanRequest.path, scanRequest.sequence])
  useEffect(() => {
    if (window.__TAURI_INTERNALS__ === undefined) return
    let active = true
    let unlisten: UnlistenFn | undefined
    void listen<{ path: string }>('storage-analyzer-open', (event) => {
      if (event.payload.path && event.payload.path.length <= 4096) setScanRequest((current) => ({ path: event.payload.path, sequence: current.sequence + 1 }))
    }).then((stop) => { if (active) unlisten = stop; else stop() })
    return () => { active = false; unlisten?.() }
  }, [])

  const close = useCallback(() => {
    void cancel().finally(() => window.__TAURI_INTERNALS__ === undefined ? window.close() : getCurrentWindow().hide())
  }, [cancel])
  useEffect(() => {
    if (window.__TAURI_INTERNALS__ === undefined) return
    let active = true
    let unlisten: UnlistenFn | undefined
    void getCurrentWindow().onCloseRequested((event) => {
      event.preventDefault()
      void cancel().finally(() => getCurrentWindow().hide())
    }).then((stop) => { if (active) unlisten = stop; else stop() })
    return () => { active = false; unlisten?.() }
  }, [cancel])
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return
      event.preventDefault()
      if (deletePreview) { setDeletePreview(null); setTypedName('') } else close()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [close, deletePreview])

  const report = status?.report
  const normalizedQuery = query.trim().toLocaleLowerCase(language)
  const nodes = useMemo(() => filterNodes(report?.children ?? [], normalizedQuery), [normalizedQuery, report?.children])
  const files = useMemo(() => filterFiles(report?.largest_files ?? [], normalizedQuery), [normalizedQuery, report?.largest_files])
  const extensions = useMemo(() => filterExtensions(report?.extensions ?? [], normalizedQuery), [normalizedQuery, report?.extensions])
  const treemapNodes = report?.children.filter((item) => item.size_bytes > 0).slice(0, 40) ?? []
  const nodeByID = new Map(treemapNodes.map((node, index) => [`${index}`, node]))
  const rectangles = squarify(treemapNodes.map((node, index) => ({ id: `${index}`, size: node.size_bytes })))

  const drill = (node: StorageScanNode) => {
    if (status && node.kind === 'directory' && node.id) void begin({ parent_scan_id: status.scan_id, node_id: node.id })
  }
  const requestDelete = async (candidate: DeleteCandidate) => {
    if (!status || deleteBusy) return
    setDeleteBusy(true); setMutationError(null); setNotice(null); setTypedName('')
    try {
      const preview = await client.call<StorageDeletePreview>('storage.delete.preview', { scan_id: status.scan_id, node_id: candidate.id })
      setDeletePreview(preview)
    } catch (reason) {
      setMutationError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setDeleteBusy(false)
    }
  }
  const confirmDelete = async () => {
    if (!status || !deletePreview || deleteBusy) return
    const refreshScanID = status.scan_id
    setDeleteBusy(true); setMutationError(null)
    try {
      const result = await client.call<StorageDeleteResult>('storage.delete.confirm', { scan_id: status.scan_id, confirmation_token: deletePreview.confirmation_token })
      setDeletePreview(null); setTypedName(''); setNotice(c.recycled(result.name))
      void begin({ refresh_scan_id: refreshScanID })
    } catch (reason) {
      setDeletePreview(null); setTypedName('')
      setMutationError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setDeleteBusy(false)
    }
  }

  return <main className="storage-analyzer-window">
    <section className="storage-analyzer" aria-labelledby="storage-analyzer-title">
      <header className="storage-analyzer__header">
        <div><span>{c.eyebrow}</span><h1 id="storage-analyzer-title"><HardDrive size={21} />{c.title}</h1><p>{status?.root ?? volumePath}</p></div>
        <Button autoFocus variant="quiet" onClick={close} aria-label={c.close}><X size={17} />{c.close}</Button>
      </header>
      <div className="storage-analyzer__body">
        {error ? <div className="storage-analyzer__state"><InlineAlert message={error} /><Button variant="primary" onClick={() => void begin({ path: volumePath })}>{c.retry}</Button></div> : null}
        {!error && (!status || status.state === 'scanning') ? <Scanning status={status} copy={c} onCancel={() => void cancel()} /> : null}
        {!error && status?.state === 'cancelled' ? <div className="storage-analyzer__state"><strong>{c.cancelled}</strong><Button variant="primary" onClick={() => void begin({ path: volumePath })}>{c.retry}</Button></div> : null}
        {!error && status?.state === 'failed' ? <div className="storage-analyzer__state"><InlineAlert message={status.error ?? c.failed} /><Button variant="primary" onClick={() => void begin({ path: volumePath })}>{c.retry}</Button></div> : null}

        {status?.state === 'complete' && report ? <div className="storage-analyzer__content">
          {mutationError ? <InlineAlert message={mutationError} /> : null}
          {notice ? <div className="success-notice" role="status">{notice}<button onClick={() => setNotice(null)} aria-label={c.dismiss}>×</button></div> : null}
          <section className="storage-analyzer__summary">
            <div><span>{c.logicalSize}</span><strong>{formatStorageBytes(report.total_bytes)}</strong></div>
            <div><span>{c.files}</span><strong>{formatInteger(report.files, language)}</strong></div>
            <div><span>{c.folders}</span><strong>{formatInteger(report.directories, language)}</strong></div>
            <div><span>{c.elapsed}</span><strong>{formatDuration(report.elapsed_ms)}</strong></div>
            <div className="storage-analyzer__actions">
              {report.parent?.id ? <Button variant="quiet" onClick={() => drill(report.parent!)}><ArrowUp size={15} />{c.up}</Button> : null}
              <Button variant="quiet" onClick={() => void begin({ refresh_scan_id: status.scan_id })}><RotateCw size={15} />{status.cached ? c.refreshCached : c.refresh}</Button>
            </div>
          </section>
          {report.partial ? <InlineAlert message={c.partial} /> : null}

          <section className="storage-treemap" aria-label={c.treemap}>
            {rectangles.map((rectangle, index) => {
              const node = nodeByID.get(rectangle.id)!
              const large = rectangle.width >= 12 && rectangle.height >= 7
              return <button type="button" key={`${node.kind}-${node.name}-${index}`} disabled={node.kind !== 'directory' || !node.id} className={`storage-tile storage-tile--${index % 8}`} style={{ left: `${rectangle.x}%`, top: `${rectangle.y / 60 * 100}%`, width: `${rectangle.width}%`, height: `${rectangle.height / 60 * 100}%` }} title={`${node.name} — ${formatStorageBytes(node.size_bytes)}`} onClick={() => drill(node)}>
                {large ? <><strong>{node.name}</strong><span>{formatStorageBytes(node.size_bytes)}</span></> : null}
              </button>
            })}
            {rectangles.length === 0 ? <span className="storage-treemap__empty">{c.empty}</span> : null}
          </section>

          <div className="storage-analyzer__toolbar">
            <div className="segmented" role="tablist" aria-label={c.views}>
              {(['folders', 'files', 'extensions'] as AnalyzerView[]).map((item) => <button type="button" role="tab" aria-selected={view === item} onClick={() => setView(item)} key={item}>{c[item]}</button>)}
            </div>
            <label className="search-field"><Search size={15} /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={c.search} aria-label={c.search} /></label>
          </div>

          <section className="storage-analyzer__list">
            {view === 'folders' ? nodes.map((node, index) => <NodeRow node={node} total={report.total_bytes} rank={index + 1} key={`${node.kind}-${node.name}`} copy={c} busy={deleteBusy} onOpen={() => drill(node)} onDelete={(candidate) => void requestDelete(candidate)} />) : null}
            {view === 'files' ? files.map((file, index) => <FileRow file={file} total={report.total_bytes} rank={index + 1} key={`${file.relative_path}-${index}`} copy={c} busy={deleteBusy} onDelete={(candidate) => void requestDelete(candidate)} />) : null}
            {view === 'extensions' ? extensions.map((extension, index) => <ExtensionRow extension={extension} total={report.total_bytes} rank={index + 1} key={`${extension.extension}-${index}`} copy={c} />) : null}
            {((view === 'folders' && nodes.length === 0) || (view === 'files' && files.length === 0) || (view === 'extensions' && extensions.length === 0)) ? <div className="storage-analyzer__empty">{c.noMatches}</div> : null}
          </section>
          <footer className="storage-analyzer__footer"><span>{c.logicalNote}</span><span>{c.skipped(report.skipped)}</span></footer>
        </div> : null}
      </div>
    </section>
    <ConfirmDialog open={deletePreview !== null} title={c.deleteTitle} description={deletePreview ? <>
      <p>{c.deleteDescription}</p>
      <div className="storage-delete-preview"><strong>{deletePreview.name}</strong><span>{c.deleteKind(deletePreview.kind)} · {formatStorageBytes(deletePreview.size_bytes)}</span></div>
      <ul><li>{c.deleteRuleRecycle}</li><li>{c.deleteRuleRescan}</li><li>{c.deleteRuleChanged}</li></ul>
      {deletePreview.requires_typed_name ? <label className="field"><span className="field__label">{c.typeName(deletePreview.name)}</span><input value={typedName} onChange={(event) => setTypedName(event.target.value)} spellCheck={false} autoComplete="off" /></label> : null}
    </> : null} confirmLabel={c.deleteConfirm} danger busy={deleteBusy} confirmDisabled={Boolean(deletePreview?.requires_typed_name && typedName !== deletePreview.name)} onCancel={() => { setDeletePreview(null); setTypedName('') }} onConfirm={() => void confirmDelete()} />
  </main>
}

function Scanning({ status, copy, onCancel }: { status: StorageScanStatus | null; copy: typeof analyzerCopy.en; onCancel: () => void }) {
  return <div className="storage-analyzer__state" aria-live="polite">
    <div className="storage-scan-spinner"><HardDrive size={29} /><span /></div>
    <div><strong>{copy.scanning}</strong><p>{status?.current_path || copy.starting}</p></div>
    <div className="storage-scan-stats"><span>{formatStorageBytes(status?.bytes_scanned ?? 0)}</span><span>{formatInteger(status?.files_scanned ?? 0, copy.locale)} {copy.filesLower}</span><span>{formatInteger(status?.directories_scanned ?? 0, copy.locale)} {copy.foldersLower}</span></div>
    <div className="storage-scan-progress"><span /></div>
    <Button variant="quiet" onClick={onCancel}>{copy.cancel}</Button>
  </div>
}

function NodeRow({ node, total, rank, copy, busy, onOpen, onDelete }: { node: StorageScanNode; total: number; rank: number; copy: typeof analyzerCopy.en; busy: boolean; onOpen: () => void; onDelete: (candidate: DeleteCandidate) => void }) {
  const details = <><strong>{node.name}</strong><small>{copy.contents(node.files, node.directories)}</small><UsageBar value={node.size_bytes} total={total} /></>
  const deletable = Boolean(node.deletable && node.id && (node.kind === 'directory' || node.kind === 'file'))
  return <div className="storage-analyzer-row"><span className="storage-rank">{rank}</span>{node.kind === 'directory' ? <Folder size={16} /> : <File size={16} />}
    {node.kind === 'directory' && node.id ? <button type="button" className="storage-row-main" onClick={onOpen}>{details}</button> : <div className="storage-row-main">{details}</div>}
    <b>{formatStorageBytes(node.size_bytes)}</b>
    {deletable ? <Button variant="quiet" className="storage-delete-button" disabled={busy} title={copy.deleteNamed(node.name)} aria-label={copy.deleteNamed(node.name)} onClick={() => onDelete({ id: node.id!, name: node.name, kind: node.kind as 'directory' | 'file', size: node.size_bytes, files: node.files, directories: node.directories })}><Trash2 size={15} /></Button> : <span />}
  </div>
}

function FileRow({ file, total, rank, copy, busy, onDelete }: { file: StorageScanFile; total: number; rank: number; copy: typeof analyzerCopy.en; busy: boolean; onDelete: (candidate: DeleteCandidate) => void }) {
  return <div className="storage-analyzer-row"><span className="storage-rank">{rank}</span><File size={16} /><div className="storage-row-main"><strong>{file.name}</strong><small>{file.relative_path}</small><UsageBar value={file.size_bytes} total={total} /></div><b>{formatStorageBytes(file.size_bytes)}</b>
    {file.deletable && file.id ? <Button variant="quiet" className="storage-delete-button" disabled={busy} title={copy.deleteNamed(file.name)} aria-label={copy.deleteNamed(file.name)} onClick={() => onDelete({ id: file.id!, name: file.name, kind: 'file', size: file.size_bytes, files: 1, directories: 0 })}><Trash2 size={15} /></Button> : <span />}
  </div>
}

function ExtensionRow({ extension, total, rank, copy }: { extension: StorageScanExtension; total: number; rank: number; copy: typeof analyzerCopy.en }) {
  return <div className="storage-analyzer-row"><span className="storage-rank">{rank}</span><span className="storage-extension-dot" /><div className="storage-row-main"><strong>{extension.extension}</strong><small>{copy.fileCount(extension.files)}</small><UsageBar value={extension.size_bytes} total={total} /></div><b>{formatStorageBytes(extension.size_bytes)}</b><span /></div>
}

function UsageBar({ value, total }: { value: number; total: number }) {
  const width = total > 0 ? Math.max(0.5, Math.min(100, value / total * 100)) : 0
  return <span className="storage-usage"><i style={{ width: `${width}%` }} /></span>
}

function filterNodes(values: StorageScanNode[], query: string): StorageScanNode[] { return query ? values.filter((value) => value.name.toLocaleLowerCase().includes(query)) : values }
function filterFiles(values: StorageScanFile[], query: string): StorageScanFile[] { return query ? values.filter((value) => `${value.name}\0${value.relative_path}`.toLocaleLowerCase().includes(query)) : values }
function filterExtensions(values: StorageScanExtension[], query: string): StorageScanExtension[] { return query ? values.filter((value) => value.extension.toLocaleLowerCase().includes(query)) : values }

export function formatStorageBytes(value: number): string {
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB']
  let size = Math.max(0, value); let unit = 0
  while (size >= 1024 && unit < units.length - 1) { size /= 1024; unit++ }
  return `${size.toFixed(unit === 0 ? 0 : size >= 100 ? 0 : 1)} ${units[unit]}`
}

function formatInteger(value: number, locale: string): string { return Math.max(0, value).toLocaleString(locale) }
function formatDuration(milliseconds: number): string { return milliseconds < 1000 ? `${milliseconds} ms` : `${(milliseconds / 1000).toFixed(1)} s` }

function pause(milliseconds: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const abort = () => { window.clearTimeout(timeout); reject(new DOMException('Aborted', 'AbortError')) }
    const timeout = window.setTimeout(() => { signal.removeEventListener('abort', abort); resolve() }, milliseconds)
    signal.addEventListener('abort', abort, { once: true })
  })
}

const analyzerCopy = {
  en: {
    locale: 'en-US', eyebrow: 'Space analyzer', title: 'What occupies this drive', close: 'Close', dismiss: 'Dismiss', retry: 'Scan again', cancel: 'Cancel scan', cancelled: 'Scan cancelled', failed: 'Scan failed', scanning: 'Reading filesystem metadata…', starting: 'Preparing the volume scan', filesLower: 'files', foldersLower: 'folders', logicalSize: 'Logical size', files: 'Files', folders: 'Folders', elapsed: 'Scan time', up: 'Up one level', refresh: 'Rescan', refreshCached: 'Refresh cached view', partial: 'The bounded time or entry limit was reached. The visible result is partial.', treemap: 'Storage treemap', empty: 'No files in this directory', views: 'Analyzer views', search: 'Filter results', noMatches: 'Nothing matches this filter.', extensions: 'Extensions', contents: (files: number, folders: number) => `${files.toLocaleString()} files · ${folders.toLocaleString()} folders`, fileCount: (files: number) => `${files.toLocaleString()} files`, logicalNote: 'Logical sizes; links, reparse points and inaccessible paths are skipped. Visited folders are cached for five minutes; Rescan always reads fresh metadata.', skipped: (count: number) => `${count.toLocaleString()} skipped entries`, deleteNamed: (name: string) => `Delete ${name}`, deleteTitle: 'Move to Recycle Bin?', deleteDescription: 'Review the exact scanned item before continuing.', deleteKind: (kind: string): string => kind === 'directory' ? 'Folder' : 'File', deleteRuleRecycle: 'The item is moved to the system Recycle Bin, never permanently deleted here.', deleteRuleRescan: 'This folder is rescanned immediately after the move.', deleteRuleChanged: 'The operation stops if the item changed, became a link or the confirmation expired.', typeName: (name: string) => `Type “${name}” to confirm this large folder`, deleteConfirm: 'Move to Recycle Bin', recycled: (name: string) => `${name} was moved to the Recycle Bin.`,
  },
  ru: {
    locale: 'ru-RU', eyebrow: 'Анализ места', title: 'Что занимает накопитель', close: 'Закрыть', dismiss: 'Скрыть', retry: 'Сканировать снова', cancel: 'Остановить', cancelled: 'Сканирование остановлено', failed: 'Ошибка сканирования', scanning: 'Чтение метаданных файловой системы…', starting: 'Подготовка сканирования тома', filesLower: 'файлов', foldersLower: 'папок', logicalSize: 'Логический размер', files: 'Файлы', folders: 'Папки', elapsed: 'Время', up: 'На уровень выше', refresh: 'Пересканировать', refreshCached: 'Обновить кеш', partial: 'Достигнут ограничитель времени или числа записей. Показан частичный результат.', treemap: 'Карта занятого места', empty: 'В этой папке нет файлов', views: 'Режимы анализатора', search: 'Фильтр результатов', noMatches: 'По фильтру ничего не найдено.', extensions: 'Расширения', contents: (files: number, folders: number) => `${files.toLocaleString('ru-RU')} файлов · ${folders.toLocaleString('ru-RU')} папок`, fileCount: (files: number) => `${files.toLocaleString('ru-RU')} файлов`, logicalNote: 'Показаны логические размеры; ссылки, reparse points и недоступные пути пропускаются. Посещённые папки кешируются на пять минут, а «Пересканировать» всегда читает свежие данные.', skipped: (count: number) => `пропущено: ${count.toLocaleString('ru-RU')}`, deleteNamed: (name: string) => `Удалить ${name}`, deleteTitle: 'Переместить в корзину?', deleteDescription: 'Проверьте выбранный элемент перед продолжением.', deleteKind: (kind: string): string => kind === 'directory' ? 'Папка' : 'Файл', deleteRuleRecycle: 'Элемент перемещается в системную корзину — без безвозвратного удаления.', deleteRuleRescan: 'После перемещения текущая папка сразу сканируется заново.', deleteRuleChanged: 'Операция остановится, если элемент изменился, стал ссылкой или подтверждение истекло.', typeName: (name: string) => `Введите «${name}» для подтверждения большой папки`, deleteConfirm: 'Переместить в корзину', recycled: (name: string) => `${name} перемещён в корзину.`,
  },
}
