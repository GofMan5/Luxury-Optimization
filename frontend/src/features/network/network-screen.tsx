import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import { Cable, Play } from 'lucide-react'
import { useBackend } from '../../app/backend-context'
import { useLanguage } from '../../app/language-context'
import type { BufferbloatPhase, BufferbloatReport, LatencyReport, NetworkInterface, UDPLatencyReport } from '../../shared/contracts/domain'
import { useBackendResource } from '../../shared/hooks/use-backend-resource'
import { Button } from '../../shared/ui/button'
import { EmptyState, InlineAlert, LoadingState } from '../../shared/ui/feedback'
import { PageHeader } from '../../shared/ui/page-header'
import { StatusDot } from '../../shared/ui/status'

type Diagnostic = 'tcp' | 'udp' | 'bufferbloat'

export default function NetworkScreen({ embedded = false }: { embedded?: boolean }) {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = networkCopy[language]
  const load = useCallback((signal: AbortSignal) => client.call<NetworkInterface[]>('network.interfaces', {}, signal), [client])
  const resource = useBackendResource(load, [load])
  const controller = useRef<AbortController | null>(null)
  const [tcpAddress, setTCPAddress] = useState('1.1.1.1:443')
  const [udpAddress, setUDPAddress] = useState('1.1.1.1:53')
  const [tcpCount, setTCPCount] = useState(10)
  const [udpCount, setUDPCount] = useState(10)
  const [duration, setDuration] = useState(3)
  const [busy, setBusy] = useState<Diagnostic | null>(null)
  const [tcpReport, setTCPReport] = useState<LatencyReport | null>(null)
  const [udpReport, setUDPReport] = useState<UDPLatencyReport | null>(null)
  const [bufferbloat, setBufferbloat] = useState<BufferbloatReport | null>(null)
  const [error, setError] = useState<string | null>(null)
  useEffect(() => () => controller.current?.abort(), [])

  const run = async <T,>(kind: Diagnostic, method: string, payload: unknown, save: (report: T) => void) => {
    controller.current?.abort()
    const request = new AbortController()
    controller.current = request
    setBusy(kind); setError(null)
    try { save(await client.call<T>(method, payload, request.signal)) }
    catch (reason) { if (!request.signal.aborted) setError(reason instanceof Error ? reason.message : String(reason)) }
    finally { if (controller.current === request) { controller.current = null; setBusy(null) } }
  }

  const submitTCP = (event: FormEvent) => { event.preventDefault(); void run('tcp', 'network.test', { address: tcpAddress, count: tcpCount, timeout_ms: 2000 }, setTCPReport) }
  const submitUDP = (event: FormEvent) => { event.preventDefault(); void run('udp', 'network.udp', { address: udpAddress, count: udpCount, timeout_ms: 2000 }, setUDPReport) }
  const submitBufferbloat = (event: FormEvent) => {
    event.preventDefault()
    void run('bufferbloat', 'network.bufferbloat', { probe_address: tcpAddress, duration_ms: duration * 1000, streams: 1 }, setBufferbloat)
  }

  if (resource.loading && !resource.data) return <div className={embedded ? 'system-pane' : 'page'}><LoadingState label={c.loading} /></div>
  return (
    <div className={embedded ? 'system-pane' : 'page'}>
      {embedded ? <div className="system-pane-heading"><div><h2>{c.title}</h2><p>{c.description}</p></div></div> : <PageHeader title={c.title} description={c.description} />}
      {resource.error ? <InlineAlert message={resource.error} onRetry={() => { client.invalidate('network.'); resource.refresh() }} /> : null}

      <section className="panel diagnostic-tool">
        <div className="panel__header"><div><h2>{c.tcpPass}</h2><p>{c.tcpDescription}</p></div></div>
        <form className="panel__body toolbar" onSubmit={submitTCP}>
          <AddressField id="tcp-address" label={c.address} value={tcpAddress} onChange={setTCPAddress} />
          <CountField id="tcp-count" label={c.samples} value={tcpCount} max={100} onChange={setTCPCount} />
          <Button variant="primary" type="submit" disabled={busy !== null}><Play size={15} />{busy === 'tcp' ? c.measuring : c.run}</Button>
        </form>
      </section>
      {tcpReport ? <LatencyMetrics report={tcpReport} copy={c} /> : null}

      <section className="panel diagnostic-tool">
        <div className="panel__header"><div><h2>{c.udpPass}</h2><p>{c.udpDescription}</p></div></div>
        <form className="panel__body toolbar" onSubmit={submitUDP}>
          <AddressField id="udp-address" label={c.dnsServer} value={udpAddress} onChange={setUDPAddress} />
          <CountField id="udp-count" label={c.samples} value={udpCount} max={50} onChange={setUDPCount} />
          <Button variant="primary" type="submit" disabled={busy !== null}><Play size={15} />{busy === 'udp' ? c.measuring : c.run}</Button>
        </form>
      </section>
      {udpReport ? <LatencyMetrics report={udpReport} copy={c} detail={`${udpReport.protocol} · ${udpReport.question}`} /> : null}

      <section className="panel diagnostic-tool">
        <div className="panel__header"><div><h2>{c.bufferbloat}</h2><p>{c.bufferbloatDescription}</p></div></div>
        <form className="panel__body toolbar" onSubmit={submitBufferbloat}>
          <AddressField id="buffer-address" label={c.probeAddress} value={tcpAddress} onChange={setTCPAddress} />
          <div className="field"><label htmlFor="buffer-duration">{c.duration}</label><input id="buffer-duration" type="number" min="2" max="15" value={duration} onChange={(event) => setDuration(Number(event.target.value))} /></div>
          <Button variant="primary" type="submit" disabled={busy !== null}><Play size={15} />{busy === 'bufferbloat' ? c.loadingTest : c.runLoaded}</Button>
        </form>
        <p className="method-note panel__body diagnostic-note">{c.dataNote}</p>
      </section>
      {bufferbloat ? <section className="metric-grid network-metrics" aria-label={c.bufferResults}>
        <Metric label={c.baseline} value={`${bufferbloat.baseline.p95_ms.toFixed(2)} ms`} detail={c.p95} />
        <PhaseMetric label={c.download} phase={bufferbloat.download} copy={c} />
        <PhaseMetric label={c.upload} phase={bufferbloat.upload} copy={c} />
      </section> : null}
      {bufferbloat?.warnings?.map((warning) => <p className="method-note" key={warning}>{warning}</p>)}
      {error ? <InlineAlert message={error} /> : null}

      <div className="section-heading"><div><h2>{c.interfaces}</h2><p>{c.interfacesDescription}</p></div></div>
      <section className="panel">
        {resource.data?.length ? <table className="data-table"><thead><tr><th>{c.networkInterface}</th><th>{c.state}</th><th>MTU</th><th>{c.addresses}</th></tr></thead><tbody>{resource.data.map((item) => <tr key={item.index}><td><span className="cell-main"><Cable size={14} /> {item.name}</span><span className="cell-sub">{c.index} {item.index}</span></td><td><StatusDot tone={item.flags.includes('up') ? 'success' : 'muted'} label={item.flags.includes('up') ? c.up : c.down} /></td><td>{item.mtu}</td><td className="path-cell">{item.addresses.join(' · ') || c.noAddress}</td></tr>)}</tbody></table> : <EmptyState title={c.empty} detail={c.emptyDetail} />}
      </section>
    </div>
  )
}

function AddressField({ id, label, value, onChange }: { id: string; label: string; value: string; onChange: (value: string) => void }) {
  return <div className="field"><label htmlFor={id}>{label}</label><input id={id} value={value} onChange={(event) => onChange(event.target.value)} spellCheck="false" /></div>
}

function CountField({ id, label, value, max, onChange }: { id: string; label: string; value: number; max: number; onChange: (value: number) => void }) {
  return <div className="field diagnostic-count"><label htmlFor={id}>{label}</label><input id={id} type="number" min="3" max={max} value={value} onChange={(event) => onChange(Number(event.target.value))} /></div>
}

function LatencyMetrics({ report, copy, detail }: { report: LatencyReport; copy: typeof networkCopy.en; detail?: string }) {
  return <section className="metric-grid network-metrics" aria-label={copy.results}><Metric label={copy.median} value={`${report.median_ms.toFixed(2)} ms`} detail={detail ?? copy.success(report.succeeded, report.attempts)} /><Metric label={copy.p95} value={`${report.p95_ms.toFixed(2)} ms`} detail={`${copy.max} ${report.max_ms.toFixed(2)} ms`} /><Metric label={copy.jitter} value={`${report.jitter_ms.toFixed(2)} ms`} detail={copy.failed(report.failed)} /></section>
}

function PhaseMetric({ label, phase, copy }: { label: string; phase: BufferbloatPhase; copy: typeof networkCopy.en }) {
  if (!phase.supported) return <Metric label={label} value={copy.skipped} detail={phase.reason ?? copy.unavailable} />
  return <Metric label={label} value={`+${phase.p95_increase_ms.toFixed(2)} ms`} detail={`${copy.rating[phase.rating ?? 'severe']} · ${phase.throughput_mbps.toFixed(1)} Mbps`} />
}

function Metric({ label, value, detail }: { label: string; value: string; detail: string }) {
  return <div className="metric"><span className="metric__label">{label}</span><strong className="metric__value">{value}</strong><span className="metric__detail">{detail}</span></div>
}

const networkCopy = {
  en: {
    loading: 'Enumerating network interfaces…', title: 'Network quality', description: 'Measure TCP, RFC 1035 UDP/DNS and loaded latency without universal MTU, DNS or adapter guesses.',
    tcpPass: 'TCP latency', tcpDescription: 'Use the same destination and sample count before and after a change.', udpPass: 'UDP DNS round-trip', udpDescription: 'Sends bounded RFC 1035 queries to an explicit DNS server; this measures that UDP path, not every game protocol.',
    address: 'Address', dnsServer: 'DNS server', samples: 'Samples', measuring: 'Measuring…', run: 'Run test', results: 'Latency results', median: 'Median', success: (ok: number, total: number) => `${ok}/${total} successful`, p95: '95th percentile', max: 'Max', jitter: 'Jitter', failed: (count: number) => `${count} failed attempts`,
    bufferbloat: 'Loaded-latency / bufferbloat', bufferbloatDescription: 'Compares idle TCP latency with bounded HTTPS download and upload load. Each unavailable phase is reported separately.', probeAddress: 'Latency probe', duration: 'Seconds per phase', loadingTest: 'Loading link…', runLoaded: 'Run loaded test', dataNote: 'Uses Cloudflare speed endpoints, one stream and at most 128 MiB per direction. Results are diagnostic and never change network settings.', bufferResults: 'Bufferbloat results', baseline: 'Idle p95', download: 'Download load', upload: 'Upload load', skipped: 'Skipped', unavailable: 'Endpoint unavailable', rating: { low: 'Low', moderate: 'Moderate', high: 'High', severe: 'Severe' },
    interfaces: 'Interfaces', interfacesDescription: 'Native interface state reported by the operating system.', networkInterface: 'Interface', state: 'State', addresses: 'Addresses', index: 'Index', up: 'Up', down: 'Down', noAddress: 'No address', empty: 'No interfaces reported', emptyDetail: 'The operating system did not expose a usable network interface.',
  },
  ru: {
    loading: 'Поиск сетевых интерфейсов…', title: 'Качество сети', description: 'Измерение TCP, UDP/DNS по RFC 1035 и задержки под нагрузкой без универсальных MTU, DNS и сомнительных твиков адаптера.',
    tcpPass: 'TCP-задержка', tcpDescription: 'До и после изменения используйте один адрес и одинаковое число измерений.', udpPass: 'UDP DNS round-trip', udpDescription: 'Отправляет ограниченные RFC 1035-запросы выбранному DNS-серверу; это тест данного UDP-пути, а не любого игрового протокола.',
    address: 'Адрес', dnsServer: 'DNS-сервер', samples: 'Измерения', measuring: 'Измерение…', run: 'Запустить', results: 'Результаты задержки', median: 'Медиана', success: (ok: number, total: number) => `успешно ${ok} из ${total}`, p95: '95-й процентиль', max: 'Максимум', jitter: 'Джиттер', failed: (count: number) => `ошибок: ${count}`,
    bufferbloat: 'Задержка под нагрузкой / bufferbloat', bufferbloatDescription: 'Сравнивает TCP-задержку без нагрузки с ограниченной HTTPS-загрузкой и отдачей. Недоступность каждого этапа показывается отдельно.', probeAddress: 'Адрес замера', duration: 'Секунд на этап', loadingTest: 'Нагрузка канала…', runLoaded: 'Запустить нагрузку', dataNote: 'Используются speed-endpoints Cloudflare, один поток и не более 128 МиБ в каждом направлении. Диагностика не меняет сетевые настройки.', bufferResults: 'Результаты bufferbloat', baseline: 'p95 без нагрузки', download: 'При загрузке', upload: 'При отдаче', skipped: 'Пропущено', unavailable: 'Endpoint недоступен', rating: { low: 'Низкая', moderate: 'Средняя', high: 'Высокая', severe: 'Критичная' },
    interfaces: 'Интерфейсы', interfacesDescription: 'Нативное состояние интерфейсов от операционной системы.', networkInterface: 'Интерфейс', state: 'Состояние', addresses: 'Адреса', index: 'Индекс', up: 'Активен', down: 'Неактивен', noAddress: 'Нет адреса', empty: 'Интерфейсы не найдены', emptyDetail: 'Операционная система не предоставила доступный сетевой интерфейс.',
  },
}
