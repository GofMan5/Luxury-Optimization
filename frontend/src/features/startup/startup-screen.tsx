import { useCallback, useMemo, useState } from 'react'
import { Search } from 'lucide-react'
import { useBackend } from '../../app/backend-context'
import { useLanguage } from '../../app/language-context'
import type { StartupEntry, StartupReport } from '../../shared/contracts/domain'
import { useBackendResource } from '../../shared/hooks/use-backend-resource'
import { Button } from '../../shared/ui/button'
import { ConfirmDialog } from '../../shared/ui/confirm-dialog'
import { EmptyState, InlineAlert, LoadingState } from '../../shared/ui/feedback'
import { PageHeader } from '../../shared/ui/page-header'
import { StatusDot } from '../../shared/ui/status'

export default function StartupScreen({ embedded = false }: { embedded?: boolean }) {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = startupCopy[language]
  const load = useCallback((signal: AbortSignal) => client.call<StartupReport>('startup.list', {}, signal), [client])
  const resource = useBackendResource(load, [load])
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<StartupEntry | null>(null)
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const entries = useMemo(() => (resource.data?.entries ?? []).filter((entry) => `${entry.name} ${entry.command} ${entry.scope}`.toLowerCase().includes(query.toLowerCase())), [resource.data, query])

  const mutate = async () => {
    if (!selected) return
    const enabled = selected.state !== 'present'
    setBusy(true)
    try {
      await client.call('startup.set', { name: selected.name, enabled })
      setMessage(c.changed(selected.name, enabled))
      setSelected(null)
      resource.refresh()
    } catch (error) { setMessage(error instanceof Error ? error.message : String(error)) }
    finally { setBusy(false) }
  }

  if (resource.loading && !resource.data) return <div className={embedded ? 'system-pane' : 'page'}><LoadingState label={c.loading} /></div>
  const search = <label className="search-field"><Search size={16} /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={c.search} aria-label={c.search} /></label>
  return (
    <div className={embedded ? 'system-pane' : 'page'}>
      {embedded ? <div className="system-pane-heading"><div><h2>{c.title}</h2><p>{c.description}</p></div>{search}</div> : <PageHeader title={c.title} description={c.description} actions={search} />}
      {resource.error ? <InlineAlert message={resource.error} onRetry={() => { client.invalidate('startup.'); resource.refresh() }} /> : null}
      {message ? <div className="success-notice" role="status">{message}<button onClick={() => setMessage(null)} aria-label={c.dismiss}>×</button></div> : null}
      <section className="panel">
        {entries.length ? <table className="data-table"><thead><tr><th>{c.entry}</th><th>{c.scope}</th><th>{c.state}</th><th>{c.command}</th><th aria-label={c.actions} /></tr></thead><tbody>{entries.map((entry) => { const mutable = entry.scope === 'HKCU' || entry.scope.toLowerCase() === 'user'; const enabled = entry.state === 'present'; return <tr key={`${entry.scope}-${entry.name}-${entry.state}`}><td><span className="cell-main">{entry.name}</span></td><td>{entry.scope}</td><td><StatusDot tone={enabled ? 'success' : 'muted'} label={enabled ? c.enabled : c.disabledByLuxury} /></td><td className="path-cell">{entry.command}</td><td><div className="table-actions"><Button variant="quiet" disabled={!mutable} title={mutable ? undefined : c.readOnly} onClick={() => setSelected(entry)}>{enabled ? c.disable : c.enable}</Button></div></td></tr>})}</tbody></table> : <EmptyState title={c.empty} detail={c.emptyDetail} />}
      </section>
      {resource.data?.warnings?.map((warning) => <InlineAlert key={warning} message={warning} />)}
      <ConfirmDialog open={selected !== null} title={c.confirmTitle(selected?.name ?? '', selected?.state === 'present')} description={<><p>{c.confirmDescription}</p><ul><li>{c.scope}: {selected?.scope}</li><li>{c.rule1}</li><li>{c.rule2}</li></ul></>} confirmLabel={selected?.state === 'present' ? c.disableEntry : c.enableEntry} busy={busy} onCancel={() => setSelected(null)} onConfirm={() => void mutate()} />
    </div>
  )
}

const startupCopy = {
  en: {
    loading: 'Reading startup entries…', title: 'Startup', description: 'Manage only reversible current-user entries. Machine-wide entries remain visible and read-only.', search: 'Filter startup', changed: (name: string, enabled: boolean) => `${name} is now ${enabled ? 'enabled' : 'disabled'}; its exact original value remains recoverable.`, dismiss: 'Dismiss notification', entry: 'Entry', scope: 'Scope', state: 'State', command: 'Command', actions: 'Actions', enabled: 'Enabled', disabledByLuxury: 'Disabled by Luxury', readOnly: 'System-wide entry is read-only', disable: 'Disable', enable: 'Enable', empty: 'No matching startup entries', emptyDetail: 'The optimizer does not invent or hide entries it cannot read.', confirmTitle: (name: string, enabled: boolean) => `${enabled ? 'Disable' : 'Enable'} ${name}?`, confirmDescription: 'The exact current-user command and registry type are preserved before disabling.', rule1: 'No process is terminated', rule2: 'The action can be reversed from this screen', disableEntry: 'Disable entry', enableEntry: 'Enable entry',
  },
  ru: {
    loading: 'Чтение автозагрузки…', title: 'Автозагрузка', description: 'Управление только обратимыми записями текущего пользователя. Системные записи видны, но доступны только для чтения.', search: 'Поиск в автозагрузке', changed: (name: string, enabled: boolean) => `${name}: ${enabled ? 'включено' : 'отключено'}. Точное исходное значение сохранено для восстановления.`, dismiss: 'Закрыть уведомление', entry: 'Программа', scope: 'Область', state: 'Состояние', command: 'Команда', actions: 'Действия', enabled: 'Включена', disabledByLuxury: 'Отключена Luxury', readOnly: 'Системная запись доступна только для чтения', disable: 'Отключить', enable: 'Включить', empty: 'Записи не найдены', emptyDetail: 'Оптимизатор не придумывает и не скрывает записи, которые не может прочитать.', confirmTitle: (name: string, enabled: boolean) => `${enabled ? 'Отключить' : 'Включить'} ${name}?`, confirmDescription: 'Перед отключением сохраняются точная команда текущего пользователя и тип registry-значения.', rule1: 'Запущенный процесс не завершается', rule2: 'Действие можно отменить на этом экране', disableEntry: 'Отключить запись', enableEntry: 'Включить запись',
  },
}
