import { useCallback, useMemo, useState } from 'react'
import { Search, Shield, Wrench } from 'lucide-react'
import { useBackend } from '../../app/backend-context'
import { useLanguage } from '../../app/language-context'
import type { MutationResult, ServiceEntry, ServicesReport } from '../../shared/contracts/domain'
import { useBackendResource } from '../../shared/hooks/use-backend-resource'
import { EmptyState, InlineAlert, LoadingState } from '../../shared/ui/feedback'
import { PageHeader } from '../../shared/ui/page-header'
import { StatusDot } from '../../shared/ui/status'
import { Button } from '../../shared/ui/button'
import { ConfirmDialog } from '../../shared/ui/confirm-dialog'

type ServiceState = 'all' | 'running' | 'stopped'
type ServiceOrigin = 'all' | 'system' | 'third-party'

export default function ServicesScreen({ embedded = false }: { embedded?: boolean }) {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = servicesCopy[language]
  const [state, setState] = useState<ServiceState>('all')
  const [origin, setOrigin] = useState<ServiceOrigin>('all')
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<ServiceEntry | null>(null)
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const load = useCallback((signal: AbortSignal) => client.call<ServicesReport>('services.list', { state, match: '' }, signal), [client, state])
  const resource = useBackendResource(load, [load])
  const allServices = resource.data?.services ?? []
  const services = useMemo(() => allServices.filter((service) => {
    const matchesOrigin = origin === 'all' || (origin === 'system' ? service.system : !service.system)
    return matchesOrigin && `${service.display_name} ${service.name} ${service.binary_path ?? ''}`.toLowerCase().includes(query.toLowerCase())
  }), [allServices, origin, query])
  const systemCount = allServices.filter((service) => service.system).length
  const runningCount = allServices.filter((service) => service.state === 'running').length

  const changeService = async () => {
    if (!selected) return
    const enabled = selected.start_type === 'disabled'
    setBusy(true); setMessage(null)
    try {
      await client.call<MutationResult>('services.set', { name: selected.name, enabled })
      setMessage(c.changed(selected.display_name || selected.name, enabled))
      setSelected(null)
      resource.refresh()
    } catch (error) { setMessage(error instanceof Error ? error.message : String(error)) }
    finally { setBusy(false) }
  }

  if (resource.loading && !resource.data) return <div className={embedded ? 'system-pane' : 'page'}><LoadingState label={c.loading} /></div>
  return (
    <div className={embedded ? 'system-pane' : 'page'}>
      {embedded ? <div className="system-pane-heading"><div><h2>{c.title}</h2><p>{c.description}</p></div></div> : <PageHeader title={c.title} description={c.description} />}
      <div className="system-summary" aria-label={c.summary}>
        <div><span>{c.visible}</span><strong>{allServices.length}</strong></div>
        <div><span>{c.running}</span><strong>{runningCount}</strong></div>
        <div><span>{c.system}</span><strong>{systemCount}</strong></div>
        <div><span>{c.thirdParty}</span><strong>{allServices.length - systemCount}</strong></div>
      </div>
      <div className="toolbar service-toolbar">
        <div className="segmented" aria-label={c.stateFilter}><button aria-pressed={state === 'all'} onClick={() => setState('all')}>{c.all}</button><button aria-pressed={state === 'running'} onClick={() => setState('running')}>{c.running}</button><button aria-pressed={state === 'stopped'} onClick={() => setState('stopped')}>{c.stopped}</button></div>
        <div className="segmented" aria-label={c.originFilter}><button aria-pressed={origin === 'all'} onClick={() => setOrigin('all')}>{c.anyOrigin}</button><button aria-pressed={origin === 'system'} onClick={() => setOrigin('system')}><Shield size={13} />{c.system}</button><button aria-pressed={origin === 'third-party'} onClick={() => setOrigin('third-party')}><Wrench size={13} />{c.thirdParty}</button></div>
        <div className="toolbar__spacer" />
        <label className="search-field"><Search size={16} /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={c.search} aria-label={c.search} /></label>
      </div>
      {resource.error ? <InlineAlert message={resource.error} onRetry={() => { client.invalidate('services.'); resource.refresh() }} /> : null}
      {message ? <div className="success-notice" role="status">{message}<button onClick={() => setMessage(null)} aria-label={c.dismiss}>×</button></div> : null}
      <section className="panel service-table-panel">
        {services.length ? <table className="data-table"><thead><tr><th>{c.service}</th><th>{c.origin}</th><th>{c.state}</th><th>{c.startType}</th><th aria-label={c.actions} /></tr></thead><tbody>{services.map((service) => { const disabled = service.start_type === 'disabled'; return <tr key={service.name}><td title={service.binary_path}><span className="cell-main">{service.display_name || service.name}</span><span className="service-description">{localizeServiceDescription(service, language, c.noDescription)}</span><span className="cell-sub">{service.name}{service.process_id ? ` · PID ${service.process_id}` : ''}</span></td><td><div className="service-badges"><span className={`origin-badge ${service.system ? 'origin-badge--system' : ''}`}>{service.system ? <Shield size={12} /> : <Wrench size={12} />}{service.system ? c.system : c.thirdParty}</span>{service.critical ? <span className="origin-badge origin-badge--critical">{c.critical}</span> : null}</div></td><td><StatusDot tone={service.state === 'running' ? 'success' : 'muted'} label={localizeState(service.state, language)} /></td><td>{localizeStartType(service.start_type, language)}</td><td><div className="table-actions"><Button variant="quiet" disabled={!service.manageable} title={service.critical && !disabled ? c.protectedTitle : undefined} onClick={() => setSelected(service)}>{service.critical && !disabled ? c.protected : disabled ? c.enable : c.disable}</Button></div></td></tr> })}</tbody></table> : <EmptyState title={c.empty} detail={c.emptyDetail} />}
      </section>
      <p className="table-footnote">{c.footnote(services.length, resource.data?.skipped ?? 0)}</p>
      <ConfirmDialog open={selected !== null} title={selected?.start_type === 'disabled' ? c.enableTitle(selected?.display_name || selected?.name || '') : c.disableTitle(selected?.display_name || selected?.name || '')} description={<><p>{selected ? localizeServiceDescription(selected, language, c.noDescription) : c.noDescription}</p>{selected?.system ? <p className="danger-copy">{c.systemWarning}</p> : null}<ul><li>{c.currentStart}: {localizeStartType(selected?.start_type ?? '', language)}</li><li>{c.dependencies}: {selected?.dependencies?.length ? selected.dependencies.join(', ') : c.none}</li><li>{c.processRule}</li><li>{c.backupRule}</li></ul></>} confirmLabel={selected?.start_type === 'disabled' ? c.enable : c.disable} danger={selected?.system && selected.start_type !== 'disabled'} busy={busy} onCancel={() => setSelected(null)} onConfirm={() => void changeService()} />
    </div>
  )
}

function localizeState(value: string, language: 'ru' | 'en'): string {
  if (language === 'en') return value
  return ({ running: 'работает', stopped: 'остановлена', pending: 'изменяется', failed: 'ошибка' } as Record<string, string>)[value] ?? value
}

function localizeStartType(value: string, language: 'ru' | 'en'): string {
  if (language === 'en') return value
  return ({ automatic: 'автоматически', manual: 'вручную', disabled: 'отключена', boot: 'загрузка', system: 'системная' } as Record<string, string>)[value] ?? value
}

function localizeServiceDescription(service: ServiceEntry, language: 'ru' | 'en', fallback: string): string {
  const known: Record<string, [string, string]> = {
    bfe: ['Controls firewall and IPsec filtering policy.', 'Управляет политиками фильтрации Firewall и IPsec.'],
    mpssvc: ['Windows Defender Firewall blocks unauthorized network access.', 'Брандмауэр Защитника Windows блокирует несанкционированный сетевой доступ.'],
    rpcss: ['Core Remote Procedure Call service required by Windows components.', 'Базовая служба удалённого вызова процедур, необходимая компонентам Windows.'],
    dcomlaunch: ['Starts DCOM and COM infrastructure used throughout Windows.', 'Запускает инфраструктуру DCOM и COM, используемую всей Windows.'],
    plugplay: ['Detects and configures hardware and device changes.', 'Обнаруживает и настраивает оборудование и изменения устройств.'],
    power: ['Coordinates system power policy and power events.', 'Управляет политикой питания и событиями энергосостояния.'],
    eventlog: ['Records Windows system, security and application events.', 'Записывает системные события, безопасность и события приложений Windows.'],
    dhcp: ['Obtains and renews IP configuration from DHCP servers.', 'Получает и обновляет IP-конфигурацию через DHCP.'],
    dnscache: ['Caches DNS names and registers the computer name.', 'Кеширует DNS-имена и регистрирует имя компьютера.'],
    windefend: ['Microsoft Defender Antivirus real-time protection service.', 'Служба антивирусной защиты Microsoft Defender в реальном времени.'],
    wuauserv: ['Detects, downloads and installs Windows updates.', 'Обнаруживает, загружает и устанавливает обновления Windows.'],
  }
  const description = known[service.name.toLowerCase()]
  return description ? description[language === 'ru' ? 1 : 0] : service.description || fallback
}

const servicesCopy = {
  en: {
    loading: 'Reading service inventory…', title: 'Services', description: 'Complete Windows service inventory with native descriptions, ownership and guarded startup controls.', summary: 'Service summary', visible: 'Loaded', running: 'Running', system: 'System', thirdParty: 'Third-party', stateFilter: 'Service state', originFilter: 'Service origin', all: 'All states', stopped: 'Stopped', anyOrigin: 'All origins', search: 'Filter services', service: 'Service and purpose', origin: 'Origin', state: 'State', startType: 'Start type', actions: 'Actions', noDescription: 'Windows did not provide a description for this service.', critical: 'Critical', protected: 'Protected', protectedTitle: 'Critical Windows services cannot be disabled', enable: 'Enable', disable: 'Disable', dismiss: 'Dismiss notification', changed: (name: string, enabled: boolean) => `${name}: startup is now ${enabled ? 'enabled' : 'disabled'} and verified.`, enableTitle: (name: string) => `Enable ${name}?`, disableTitle: (name: string) => `Disable ${name}?`, systemWarning: 'This is a Windows system service. Disabling it can break dependent features; review its purpose and dependencies first.', currentStart: 'Current startup', dependencies: 'Dependencies', none: 'None reported', processRule: 'The currently running process is not terminated', backupRule: 'The exact original startup type is backed up and restored on Enable', empty: 'No matching services', emptyDetail: 'Change the filters or search. Unavailable service managers fail safely.', footnote: (visible: number, skipped: number) => `${visible} visible · ${skipped} inaccessible entries skipped`,
  },
  ru: {
    loading: 'Чтение списка служб…', title: 'Службы', description: 'Полный список служб Windows с нативным описанием, типом и защищённым управлением запуска.', summary: 'Сводка служб', visible: 'Загружено', running: 'Работают', system: 'Системные', thirdParty: 'Сторонние', stateFilter: 'Состояние служб', originFilter: 'Происхождение служб', all: 'Все состояния', stopped: 'Остановлены', anyOrigin: 'Все типы', search: 'Поиск служб', service: 'Служба и назначение', origin: 'Тип', state: 'Состояние', startType: 'Запуск', actions: 'Действия', noDescription: 'Windows не предоставила описание этой службы.', critical: 'Критическая', protected: 'Защищена', protectedTitle: 'Критические службы Windows нельзя отключить', enable: 'Включить', disable: 'Отключить', dismiss: 'Закрыть уведомление', changed: (name: string, enabled: boolean) => `${name}: запуск ${enabled ? 'включён' : 'отключён'} и проверен.`, enableTitle: (name: string) => `Включить ${name}?`, disableTitle: (name: string) => `Отключить ${name}?`, systemWarning: 'Это системная служба Windows. Отключение может сломать зависимые функции — сначала проверьте назначение и зависимости.', currentStart: 'Текущий запуск', dependencies: 'Зависимости', none: 'Не указаны', processRule: 'Текущий процесс службы не завершается', backupRule: 'Точный исходный тип запуска сохраняется и возвращается при включении', empty: 'Службы не найдены', emptyDetail: 'Измените фильтры или поиск. Недоступный диспетчер служб безопасно пропускается.', footnote: (visible: number, skipped: number) => `Показано: ${visible} · недоступно и пропущено: ${skipped}`,
  },
}
