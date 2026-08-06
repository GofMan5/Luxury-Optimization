import { useCallback, useState, type FormEvent } from 'react'
import { Cable, Play } from 'lucide-react'
import { useBackend } from '../../app/backend-context'
import { useLanguage } from '../../app/language-context'
import type { LatencyReport, NetworkInterface } from '../../shared/contracts/domain'
import { useBackendResource } from '../../shared/hooks/use-backend-resource'
import { Button } from '../../shared/ui/button'
import { EmptyState, InlineAlert, LoadingState } from '../../shared/ui/feedback'
import { PageHeader } from '../../shared/ui/page-header'
import { StatusDot } from '../../shared/ui/status'

export default function NetworkScreen({ embedded = false }: { embedded?: boolean }) {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = networkCopy[language]
  const load = useCallback((signal: AbortSignal) => client.call<NetworkInterface[]>('network.interfaces', {}, signal), [client])
  const resource = useBackendResource(load, [load])
  const [address, setAddress] = useState('1.1.1.1:443')
  const [count, setCount] = useState(10)
  const [testing, setTesting] = useState(false)
  const [report, setReport] = useState<LatencyReport | null>(null)
  const [error, setError] = useState<string | null>(null)

  const run = async (event: FormEvent) => {
    event.preventDefault()
    setTesting(true); setError(null)
    try { setReport(await client.call<LatencyReport>('network.test', { address, count, timeout_ms: 2000 })) }
    catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)) }
    finally { setTesting(false) }
  }

  if (resource.loading && !resource.data) return <div className={embedded ? 'system-pane' : 'page'}><LoadingState label={c.loading} /></div>
  return (
    <div className={embedded ? 'system-pane' : 'page'}>
      {embedded ? <div className="system-pane-heading"><div><h2>{c.title}</h2><p>{c.description}</p></div></div> : <PageHeader title={c.title} description={c.description} />}
      {resource.error ? <InlineAlert message={resource.error} onRetry={() => { client.invalidate('network.'); resource.refresh() }} /> : null}
      <section className="panel latency-tool">
        <div className="panel__header"><div><h2>{c.tcpPass}</h2><p>{c.tcpDescription}</p></div></div>
        <form className="panel__body toolbar" onSubmit={(event) => void run(event)}>
          <div className="field"><label htmlFor="network-address">{c.address}</label><input id="network-address" value={address} onChange={(event) => setAddress(event.target.value)} spellCheck="false" /></div>
          <div className="field"><label htmlFor="network-count">{c.samples}</label><input id="network-count" type="number" min="3" max="100" value={count} onChange={(event) => setCount(Number(event.target.value))} /></div>
          <Button variant="primary" type="submit" disabled={testing}><Play size={15} />{testing ? c.measuring : c.run}</Button>
        </form>
      </section>
      {error ? <InlineAlert message={error} /> : null}
      {report ? <section className="metric-grid network-metrics" aria-label={c.results}><Metric label={c.median} value={`${report.median_ms.toFixed(2)} ms`} detail={c.success(report.succeeded, report.attempts)} /><Metric label={c.p95} value={`${report.p95_ms.toFixed(2)} ms`} detail={`${c.max} ${report.max_ms.toFixed(2)} ms`} /><Metric label={c.jitter} value={`${report.jitter_ms.toFixed(2)} ms`} detail={c.failed(report.failed)} /></section> : null}

      <div className="section-heading"><div><h2>{c.interfaces}</h2><p>{c.interfacesDescription}</p></div></div>
      <section className="panel">
        {resource.data?.length ? <table className="data-table"><thead><tr><th>{c.networkInterface}</th><th>{c.state}</th><th>MTU</th><th>{c.addresses}</th></tr></thead><tbody>{resource.data.map((item) => <tr key={item.index}><td><span className="cell-main"><Cable size={14} /> {item.name}</span><span className="cell-sub">{c.index} {item.index}</span></td><td><StatusDot tone={item.flags.includes('up') ? 'success' : 'muted'} label={item.flags.includes('up') ? c.up : c.down} /></td><td>{item.mtu}</td><td className="path-cell">{item.addresses.join(' · ') || c.noAddress}</td></tr>)}</tbody></table> : <EmptyState title={c.empty} detail={c.emptyDetail} />}
      </section>
    </div>
  )
}

function Metric({ label, value, detail }: { label: string; value: string; detail: string }) {
  return <div className="metric"><span className="metric__label">{label}</span><strong className="metric__value">{value}</strong><span className="metric__detail">{detail}</span></div>
}

const networkCopy = {
  en: {
    loading: 'Enumerating network interfaces…', title: 'Network', description: 'Measure TCP latency and inspect interfaces without universal MTU, DNS or adapter guesses.', tcpPass: 'TCP latency pass', tcpDescription: 'Use the same destination and sample count before and after a change.', address: 'Address', samples: 'Samples', measuring: 'Measuring…', run: 'Run test', results: 'Latency results', median: 'Median', success: (ok: number, total: number) => `${ok}/${total} successful`, p95: '95th percentile', max: 'Max', jitter: 'Jitter', failed: (count: number) => `${count} failed attempts`, interfaces: 'Interfaces', interfacesDescription: 'Native interface state reported by the operating system.', networkInterface: 'Interface', state: 'State', addresses: 'Addresses', index: 'Index', up: 'Up', down: 'Down', noAddress: 'No address', empty: 'No interfaces reported', emptyDetail: 'The operating system did not expose a usable network interface.',
  },
  ru: {
    loading: 'Поиск сетевых интерфейсов…', title: 'Сеть', description: 'Измерение TCP-задержки и просмотр интерфейсов без универсальных MTU, DNS и сомнительных твиков адаптера.', tcpPass: 'Проверка TCP-задержки', tcpDescription: 'До и после изменения используйте один адрес и одинаковое число измерений.', address: 'Адрес', samples: 'Измерения', measuring: 'Измерение…', run: 'Запустить', results: 'Результаты задержки', median: 'Медиана', success: (ok: number, total: number) => `успешно ${ok} из ${total}`, p95: '95-й процентиль', max: 'Максимум', jitter: 'Джиттер', failed: (count: number) => `ошибок: ${count}`, interfaces: 'Интерфейсы', interfacesDescription: 'Нативное состояние интерфейсов от операционной системы.', networkInterface: 'Интерфейс', state: 'Состояние', addresses: 'Адреса', index: 'Индекс', up: 'Активен', down: 'Неактивен', noAddress: 'Нет адреса', empty: 'Интерфейсы не найдены', emptyDetail: 'Операционная система не предоставила доступный сетевой интерфейс.',
  },
}
