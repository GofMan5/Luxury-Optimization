import { useCallback, useState } from 'react'
import { ArchiveRestore, FileClock, MonitorUp, ShieldCheck } from 'lucide-react'
import { useBackend } from '../../app/backend-context'
import { useLanguage } from '../../app/language-context'
import type { BackupSummary, MutationResult, SystemRestorePoint } from '../../shared/contracts/domain'
import { useBackendResource } from '../../shared/hooks/use-backend-resource'
import { Button } from '../../shared/ui/button'
import { ConfirmDialog } from '../../shared/ui/confirm-dialog'
import { EmptyState, InlineAlert, LoadingState } from '../../shared/ui/feedback'
import { PageHeader } from '../../shared/ui/page-header'
import { StatusDot } from '../../shared/ui/status'

type RestoreMode = 'windows' | 'luxury'

export default function RestoreScreen() {
  const { language } = useLanguage()
  const c = restoreCopy[language]
  const [mode, setMode] = useState<RestoreMode>('luxury')
  return (
    <div className="page">
      <PageHeader title={c.title} description={c.description} />
      <div className="restore-tabs" role="tablist" aria-label={c.choice}>
        <button role="tab" aria-selected={mode === 'luxury'} onClick={() => setMode('luxury')}><FileClock size={20} /><span><strong>{c.luxury}</strong><small>{c.luxuryHint}</small></span></button>
        <button role="tab" aria-selected={mode === 'windows'} onClick={() => setMode('windows')}><MonitorUp size={20} /><span><strong>{c.windows}</strong><small>{c.windowsHint}</small></span></button>
      </div>
      {mode === 'luxury' ? <LuxuryRestorePanel /> : <WindowsRestorePanel />}
    </div>
  )
}

function LuxuryRestorePanel() {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = restoreCopy[language]
  const load = useCallback((signal: AbortSignal) => client.call<BackupSummary[]>('backups.list', {}, signal), [client])
  const resource = useBackendResource(load, [load])
  const [selected, setSelected] = useState<BackupSummary | 'latest' | null>(null)
  const [restoring, setRestoring] = useState(false)
  const [result, setResult] = useState<string | null>(null)

  const restore = async () => {
    if (!selected) return
    setRestoring(true); setResult(null)
    try {
      const operation = await client.call<MutationResult>('optimization.restore', { id: selected === 'latest' ? '' : selected.id })
      setResult(operation.message)
      setSelected(null)
      resource.refresh()
    } catch (error) { setResult(error instanceof Error ? error.message : String(error)) }
    finally { setRestoring(false) }
  }

  if (resource.loading && !resource.data && !resource.error) return <LoadingState label={c.loadingLuxury} />
  return (
    <div className="restore-panel">
      {resource.error ? <InlineAlert message={`${resource.error}. ${c.uacFallback}`} onRetry={() => { client.invalidate('backups.'); resource.refresh() }} /> : null}
      {result ? <div className="success-notice" role="status"><ShieldCheck size={18} />{result}<button onClick={() => setResult(null)} aria-label={c.dismiss}>×</button></div> : null}
      <section className="restore-intro panel panel--gold"><div><ShieldCheck size={26} /><div><h2>{c.fileTitle}</h2><p>{c.fileDescription}</p></div></div><Button variant="secondary" onClick={() => setSelected('latest')}><ArchiveRestore size={16} />{c.restoreLatest}</Button></section>
      <div className="section-heading"><div><h2>{c.files}</h2><p>{c.filesDescription}</p></div></div>
      <section className="panel">
        {resource.data?.length ? <table className="data-table"><thead><tr><th>{c.created}</th><th>{c.profile}</th><th>{c.state}</th><th>{c.fileID}</th><th aria-label={c.actions} /></tr></thead><tbody>{resource.data.map((backup) => <tr key={backup.id}><td><span className="cell-main">{new Date(backup.created_at).toLocaleString(language === 'ru' ? 'ru-RU' : 'en-US')}</span></td><td>{backup.profile}</td><td><StatusDot tone={backup.restorable ? 'success' : 'muted'} label={backup.status} /></td><td className="path-cell">{backup.id}</td><td><div className="table-actions"><Button variant="quiet" disabled={!backup.restorable} onClick={() => setSelected(backup)}>{c.restore}</Button></div></td></tr>)}</tbody></table> : <EmptyState title={resource.error ? c.catalogProtected : c.noFiles} detail={resource.error ? c.catalogProtectedDetail : c.noFilesDetail} />}
      </section>
      <ConfirmDialog open={selected !== null} title={c.confirmLuxury} description={<><p>{selected === 'latest' ? c.latestDescription : c.selectedDescription(selected?.id ?? '')}</p><ul><li>{c.fileRule1}</li><li>{c.fileRule2}</li><li>{c.fileRule3}</li></ul></>} confirmLabel={c.restoreNow} busy={restoring} onCancel={() => setSelected(null)} onConfirm={() => void restore()} />
    </div>
  )
}

function WindowsRestorePanel() {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = restoreCopy[language]
  const load = useCallback((signal: AbortSignal) => client.call<SystemRestorePoint[]>('restore.system_points', {}, signal), [client])
  const resource = useBackendResource(load, [load])
  const [opening, setOpening] = useState(false)
  const [message, setMessage] = useState<string | null>(null)

  const open = async () => {
    setOpening(true); setMessage(null)
    try {
      await client.call<MutationResult>('restore.open_system', {})
      setMessage(c.windowsOpened)
    } catch (error) { setMessage(error instanceof Error ? error.message : String(error)) }
    finally { setOpening(false) }
  }

  if (resource.loading && !resource.data) return <LoadingState label={c.loadingWindows} />
  return (
    <div className="restore-panel">
      {resource.error ? <InlineAlert message={resource.error} onRetry={() => { client.invalidate('restore.system_points'); resource.refresh() }} /> : null}
      {message ? <div className="success-notice" role="status"><MonitorUp size={18} />{message}<button onClick={() => setMessage(null)} aria-label={c.dismiss}>×</button></div> : null}
      <section className="restore-intro panel panel--gold"><div><MonitorUp size={26} /><div><h2>{c.windowsTitle}</h2><p>{c.windowsDescription}</p></div></div><Button variant="primary" disabled={opening} onClick={() => void open()}>{opening ? c.opening : c.openWindows}</Button></section>
      <div className="section-heading"><div><h2>{c.systemPoints}</h2><p>{c.systemPointsDescription}</p></div></div>
      <section className="panel">
        {resource.data?.length ? <table className="data-table"><thead><tr><th>{c.created}</th><th>{c.descriptionColumn}</th><th>{c.sequence}</th></tr></thead><tbody>{resource.data.map((point) => <tr key={point.sequence_number}><td><span className="cell-main">{new Date(point.created_at).toLocaleString(language === 'ru' ? 'ru-RU' : 'en-US')}</span></td><td>{point.description || c.noDescription}</td><td>#{point.sequence_number}</td></tr>)}</tbody></table> : <EmptyState title={c.noSystemPoints} detail={c.noSystemPointsDetail} />}
      </section>
    </div>
  )
}

const restoreCopy = {
  en: {
    title: 'Restore', description: 'Choose Windows System Restore or an exact Luxury Optimization recovery file.', choice: 'Restore method', luxury: 'Luxury recovery file', luxuryHint: 'Registry and every optimizer change', windows: 'Windows System Restore', windowsHint: 'Native operating-system restore points', loadingLuxury: 'Opening Luxury recovery catalog…', loadingWindows: 'Reading Windows restore points…', uacFallback: 'Restore latest remains available and will request UAC.', dismiss: 'Dismiss notification', fileTitle: 'Exact optimizer rollback', fileDescription: 'Each sealed recovery file stores original Registry Editor, mouse, Ethernet and power state. Restore also re-enables startup entries and restores service startup types changed by Luxury Optimization.', restoreLatest: 'Restore latest file', files: 'Luxury recovery files', filesDescription: 'Created before profile application and validated before restore.', created: 'Created', profile: 'Profile', state: 'State', fileID: 'Recovery file ID', actions: 'Actions', restore: 'Restore', catalogProtected: 'Catalog requires elevation', catalogProtectedDetail: 'The protected catalog is not exposed to the WebView. Restore latest crosses UAC only for the bounded rollback.', noFiles: 'No Luxury recovery files', noFilesDetail: 'Create a local restore point from Tweaks → Profiles before applying a preset.', confirmLuxury: 'Restore every optimizer change?', latestDescription: 'The latest restorable Luxury file for this user will be selected after elevation.', selectedDescription: (id: string) => `Recovery file ${id} will be restored.`, fileRule1: 'Original registry values, mouse, Ethernet and power', fileRule2: 'Startup entries disabled by Luxury', fileRule3: 'Original service startup types plus native read-back', restoreNow: 'Restore from file', windowsOpened: 'Windows System Restore opened.', windowsTitle: 'Native Windows recovery', windowsDescription: 'The standard Microsoft wizard performs the operating-system restore. Luxury Optimization only opens rstrui.exe and does not reimplement it.', opening: 'Opening…', openWindows: 'Open Windows Restore', systemPoints: 'System restore points', systemPointsDescription: 'Read-only list from Windows System Protection.', descriptionColumn: 'Description', sequence: 'Sequence', noDescription: 'No description', noSystemPoints: 'No system restore points found', noSystemPointsDetail: 'System Protection may be disabled or Windows has not created a restore point yet.',
  },
  ru: {
    title: 'Восстановление', description: 'Выберите штатное восстановление Windows или точный файл восстановления Luxury Optimization.', choice: 'Способ восстановления', luxury: 'Файл восстановления Luxury', luxuryHint: 'Registry и все изменения оптимизатора', windows: 'Восстановление Windows', windowsHint: 'Штатные системные точки', loadingLuxury: 'Открытие файлов восстановления Luxury…', loadingWindows: 'Чтение системных точек Windows…', uacFallback: 'Восстановление последнего файла доступно через UAC.', dismiss: 'Закрыть уведомление', fileTitle: 'Точный откат изменений софта', fileDescription: 'Sealed-файл хранит исходные Registry, мышь, Ethernet и питание. Восстановление также возвращает отключённую автозагрузку и исходные типы запуска служб, изменённых Luxury.', restoreLatest: 'Последний файл', files: 'Файлы восстановления Luxury', filesDescription: 'Создаются до применения профиля и проверяются перед восстановлением.', created: 'Создан', profile: 'Профиль', state: 'Состояние', fileID: 'ID файла', actions: 'Действия', restore: 'Восстановить', catalogProtected: 'Каталог защищён правами администратора', catalogProtectedDetail: 'Защищённые данные не передаются в WebView. Последний файл восстанавливается через ограниченный UAC-откат.', noFiles: 'Файлов восстановления Luxury нет', noFilesDetail: 'Создайте локальную точку в ТВИКИ → Профили перед применением пресета.', confirmLuxury: 'Вернуть все изменения оптимизатора?', latestDescription: 'После повышения прав будет выбран последний доступный файл этого пользователя.', selectedDescription: (id: string) => `Будет применён файл восстановления ${id}.`, fileRule1: 'Исходные Registry, мышь, Ethernet и питание', fileRule2: 'Автозагрузка, отключённая Luxury', fileRule3: 'Исходные типы запуска служб и нативный read-back', restoreNow: 'Восстановить из файла', windowsOpened: 'Открыто штатное восстановление Windows.', windowsTitle: 'Штатное восстановление Windows', windowsDescription: 'Восстановление ОС выполняет стандартный мастер Microsoft. Luxury Optimization только открывает rstrui.exe и не подменяет системный механизм.', opening: 'Открытие…', openWindows: 'Открыть восстановление Windows', systemPoints: 'Системные точки', systemPointsDescription: 'Список только для чтения из защиты системы Windows.', descriptionColumn: 'Описание', sequence: 'Номер', noDescription: 'Без описания', noSystemPoints: 'Системные точки не найдены', noSystemPointsDetail: 'Защита системы может быть отключена либо Windows ещё не создала точку восстановления.',
  },
}
