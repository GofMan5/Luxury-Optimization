import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import { HardDrive, Play } from 'lucide-react'
import { emitTo } from '@tauri-apps/api/event'
import { WebviewWindow } from '@tauri-apps/api/webviewWindow'
import { useBackend } from '../../app/backend-context'
import { useLanguage } from '../../app/language-context'
import type { StoragePathReport, StorageVolumesReport } from '../../shared/contracts/domain'
import { useBackendResource } from '../../shared/hooks/use-backend-resource'
import { Button } from '../../shared/ui/button'
import { EmptyState, InlineAlert, LoadingState } from '../../shared/ui/feedback'
import { StatusDot } from '../../shared/ui/status'
import { formatStorageBytes } from './storage-analyzer'

export default function StorageScreen() {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = storageCopy[language]
  const load = useCallback((signal: AbortSignal) => client.call<StorageVolumesReport>('storage.volumes', {}, signal), [client])
  const resource = useBackendResource(load, [load])
  const controller = useRef<AbortController | null>(null)
  const [path, setPath] = useState('')
  const [sizeMB, setSizeMB] = useState(64)
  const [testing, setTesting] = useState(false)
  const [report, setReport] = useState<StoragePathReport | null>(null)
  const [error, setError] = useState<string | null>(null)
  useEffect(() => () => controller.current?.abort(), [])
  useEffect(() => {
    if (!path) setPath(resource.data?.volumes.find((volume) => !volume.read_only && volume.kind !== 'remote')?.path ?? '')
  }, [path, resource.data])

  const run = async (event: FormEvent) => {
    event.preventDefault()
    controller.current?.abort()
    const request = new AbortController()
    controller.current = request
    setTesting(true); setError(null)
    try { setReport(await client.call<StoragePathReport>('storage.test', { path, size_mb: sizeMB, block_kb: 1024 }, request.signal)) }
    catch (reason) { if (!request.signal.aborted) setError(reason instanceof Error ? reason.message : String(reason)) }
    finally { if (controller.current === request) { controller.current = null; setTesting(false) } }
  }

  if (resource.loading && !resource.data) return <div className="system-pane"><LoadingState label={c.loading} /></div>
  return <div className="system-pane">
    <div className="system-pane-heading"><div><h2>{c.title}</h2><p>{c.description}</p></div></div>
    {resource.error ? <InlineAlert message={resource.error} onRetry={() => { client.invalidate('storage.'); resource.refresh() }} /> : null}
    <section className="panel diagnostic-tool">
      <div className="panel__header"><div><h2>{c.pathProbe}</h2><p>{c.pathProbeDescription}</p></div></div>
      <form className="panel__body toolbar" onSubmit={(event) => void run(event)}>
        <div className="field storage-path-field"><label htmlFor="storage-path">{c.path}</label><input id="storage-path" list="storage-volumes" value={path} onChange={(event) => setPath(event.target.value)} spellCheck="false" /><datalist id="storage-volumes">{resource.data?.volumes.map((volume) => <option value={volume.path} key={volume.path}>{volume.name || volume.file_system}</option>)}</datalist></div>
        <div className="field"><label htmlFor="storage-size">{c.size}</label><select id="storage-size" value={sizeMB} onChange={(event) => setSizeMB(Number(event.target.value))}><option value="32">32 MiB</option><option value="64">64 MiB</option><option value="128">128 MiB</option></select></div>
        <Button variant="primary" type="submit" disabled={testing || !path}><Play size={15} />{testing ? c.testing : c.run}</Button>
      </form>
      <p className="method-note panel__body diagnostic-note">{c.method}</p>
    </section>
    {error ? <InlineAlert message={error} /> : null}
    {report ? <><section className="metric-grid network-metrics" aria-label={c.results}>
      <Metric label={c.durableWrite} value={`${report.durable_write_mb_s.toFixed(1)} MB/s`} detail={`${c.sync} ${report.sync_ms.toFixed(1)} ms`} />
      <Metric label={c.bufferedRead} value={`${report.buffered_read_mb_s.toFixed(1)} MB/s`} detail={`${formatStorageBytes(report.size_bytes)} · ${formatStorageBytes(report.block_bytes)} ${c.blocks}`} />
      <Metric label={c.integrity} value={report.verified && report.temporary_file_removed ? c.verified : c.failed} detail={report.sha256.slice(0, 16)} />
    </section><p className="method-note">{c.samePath}</p></> : null}
    {resource.data?.warnings?.map((warning) => <p className="method-note" key={warning}>{warning}</p>)}

    <div className="section-heading"><div><h2>{c.volumes}</h2><p>{c.volumesDescription}</p></div></div>
    <section className="panel">
      {resource.data?.volumes.length ? <table className="data-table storage-volume-table"><thead><tr><th>{c.volume}</th><th>{c.fileSystem}</th><th>{c.free}</th><th>{c.state}</th><th>{c.action}</th></tr></thead><tbody>{resource.data.volumes.map((volume) => <tr key={volume.path}><td><span className="cell-main"><HardDrive size={14} /> {volume.path}</span><span className="cell-sub">{volume.name || volume.kind}</span></td><td>{volume.file_system}</td><td>{formatStorageBytes(volume.available_bytes)} / {formatStorageBytes(volume.total_bytes)}</td><td><StatusDot tone={volume.read_only ? 'warning' : 'success'} label={volume.read_only ? c.readOnly : c.writable} /></td><td><Button variant="quiet" onClick={() => void openStorageAnalyzer(volume.path).catch((reason) => setError(reason instanceof Error ? reason.message : String(reason)))}>{c.analyze}</Button></td></tr>)}</tbody></table> : <EmptyState title={c.empty} detail={c.emptyDetail} />}
    </section>
  </div>
}

async function openStorageAnalyzer(path: string): Promise<void> {
  const url = `index.html?storage-analyzer=${encodeURIComponent(path)}`
  if (window.__TAURI_INTERNALS__ === undefined) {
    if (!window.open(url, 'storage-analyzer', 'popup,width=1240,height=850')) throw new Error('The analyzer window was blocked.')
    return
  }
  const existing = await WebviewWindow.getByLabel('storage-analyzer')
  if (existing) {
    await existing.show()
    await emitTo('storage-analyzer', 'storage-analyzer-open', { path })
    await existing.setFocus()
    return
  }
  const analyzer = new WebviewWindow('storage-analyzer', { url, title: 'Luxury Storage Analyzer', width: 1240, height: 850, minWidth: 760, minHeight: 560, center: true, resizable: true })
  await new Promise<void>((resolve, reject) => {
    void analyzer.once('tauri://created', () => resolve())
    void analyzer.once('tauri://error', (event) => reject(new Error(String(event.payload))))
  })
}

function Metric({ label, value, detail }: { label: string; value: string; detail: string }) {
  return <div className="metric"><span className="metric__label">{label}</span><strong className="metric__value">{value}</strong><span className="metric__detail">{detail}</span></div>
}

const storageCopy = {
  en: {
    loading: 'Enumerating storage volumes…', title: 'Storage', description: 'Inspect local filesystems, analyze what occupies them and run one bounded path probe without raw-disk access.', pathProbe: 'Filesystem path probe', pathProbeDescription: 'Creates one uniquely named temporary file, writes, syncs, reads back, verifies SHA-256 and removes it.', path: 'Existing directory', size: 'Temporary size', testing: 'Testing path…', run: 'Run path test', method: 'Buffered throughput is affected by OS cache. Compare only the same path, size and power/thermal state; this is not a vendor SSD benchmark.', results: 'Storage path results', durableWrite: 'Write + sync', bufferedRead: 'Buffered read', sync: 'fsync', blocks: 'blocks', integrity: 'Read-back', verified: 'Verified + removed', failed: 'Check failed', samePath: 'Use repeated same-path runs to detect large regressions; small differences are normal cache variance.', volumes: 'Local volumes', volumesDescription: 'Open the built-in analyzer to see large folders, files and extensions. Remote and optical drives are skipped.', volume: 'Volume', fileSystem: 'Filesystem', free: 'Available / total', state: 'State', action: 'Analyzer', analyze: 'Analyze space', readOnly: 'Read-only', writable: 'Writable', empty: 'No local volumes reported', emptyDetail: 'The operating system did not expose a supported local filesystem.',
  },
  ru: {
    loading: 'Поиск накопителей…', title: 'Накопители', description: 'Просмотр локальных файловых систем, занятого места и ограниченная проверка пути без raw-доступа к диску.', pathProbe: 'Проверка пути файловой системы', pathProbeDescription: 'Создаёт один уникально названный временный файл, записывает, синхронизирует, читает, сверяет SHA-256 и удаляет его.', path: 'Существующий каталог', size: 'Размер временного файла', testing: 'Проверка пути…', run: 'Запустить проверку', method: 'Buffered-скорость зависит от кэша ОС. Сравнивайте только одинаковый путь, размер и состояние питания/температур; это не паспортный benchmark SSD.', results: 'Результаты накопителя', durableWrite: 'Запись + sync', bufferedRead: 'Buffered-чтение', sync: 'fsync', blocks: 'блоки', integrity: 'Read-back', verified: 'Проверено и удалено', failed: 'Ошибка проверки', samePath: 'Повторяйте тест на одном пути для поиска крупных регрессий; небольшая разница из-за кэша нормальна.', volumes: 'Локальные тома', volumesDescription: 'Откройте встроенный анализатор крупных папок, файлов и расширений. Сетевые и оптические диски пропускаются.', volume: 'Том', fileSystem: 'Файловая система', free: 'Доступно / всего', state: 'Состояние', action: 'Анализатор', analyze: 'Что занимает место', readOnly: 'Только чтение', writable: 'Доступен', empty: 'Локальные тома не найдены', emptyDetail: 'Операционная система не предоставила поддерживаемую локальную файловую систему.',
  },
}
