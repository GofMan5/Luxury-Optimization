import { useEffect, useMemo, useRef, useState } from 'react'
import { Activity, ArrowRight, Gauge, RefreshCw } from 'lucide-react'
import { useBackend } from '../../app/backend-context'
import { useLanguage } from '../../app/language-context'
import type { BackgroundProcess, BackgroundReport } from '../../shared/contracts/domain'
import { Button } from '../../shared/ui/button'
import { EmptyState, InlineAlert } from '../../shared/ui/feedback'

type TargetSection = 'startup' | 'services'
type ProcessFilter = 'active' | 'all'

export default function BackgroundScreen({ onSection }: { onSection: (section: TargetSection) => void }) {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = backgroundCopy[language]
  const controller = useRef<AbortController | null>(null)
  const [report, setReport] = useState<BackgroundReport | null>(null)
  const [filter, setFilter] = useState<ProcessFilter>('active')
  const [measuring, setMeasuring] = useState(false)
  const [error, setError] = useState<string | null>(null)
  useEffect(() => () => { controller.current?.abort(); controller.current = null }, [])

  const measure = async () => {
    controller.current?.abort()
    const request = new AbortController()
    controller.current = request
    setMeasuring(true); setError(null)
    try { setReport(await client.call<BackgroundReport>('advisor.background', { sample_ms: 1500 }, request.signal)) }
    catch (reason) { if (!request.signal.aborted) setError(reason instanceof Error ? reason.message : String(reason)) }
    finally { if (controller.current === request) { controller.current = null; setMeasuring(false) } }
  }

  const processes = useMemo(() => (report?.processes ?? []).filter((process) => filter === 'all' || process.impact !== 'low' || process.advice === 'review_startup' || process.advice === 'review_service'), [report, filter])
  return (
    <div className="system-pane">
      <div className="system-pane-heading"><div><h2>{c.title}</h2><p>{c.description}</p></div><Button variant="primary" disabled={measuring} onClick={() => void measure()}>{measuring ? <Activity className="spinner" size={16} /> : <RefreshCw size={16} />}{measuring ? c.measuring : report ? c.measureAgain : c.measure}</Button></div>
      {error ? <InlineAlert message={error} onRetry={() => void measure()} /> : null}
      {report ? <>
        <section className="system-summary" aria-label={c.summary} aria-live="polite">
          <div><span>{c.sample}</span><strong>{(report.sample_ms / 1000).toFixed(2)} s</strong></div>
          <div><span>{c.measured}</span><strong>{report.measured_processes}</strong></div>
          <div><span>{c.cpu}</span><strong>{report.observed_cpu_percent.toFixed(1)}%</strong></div>
          <div><span>{c.correlated}</span><strong>{report.correlated_processes}</strong></div>
        </section>
        <div className="toolbar background-toolbar"><div className="segmented" aria-label={c.filter}><button aria-pressed={filter === 'active'} onClick={() => setFilter('active')}>{c.active}</button><button aria-pressed={filter === 'all'} onClick={() => setFilter('all')}>{c.all}</button></div><span>{c.io(report.read_mb_s, report.write_mb_s)}</span></div>
        <section className="panel background-table-panel">
          {processes.length ? <table className="data-table background-table"><thead><tr><th>{c.process}</th><th>{c.activity}</th><th>{c.memory}</th><th>{c.links}</th><th>{c.result}</th></tr></thead><tbody>{processes.map((process) => <ProcessRow key={`${process.pid}-${process.name}`} process={process} onSection={onSection} />)}</tbody></table> : <EmptyState title={c.noActive} detail={c.noActiveDetail} />}
        </section>
        <p className="method-note">{c.method(report.logical_processors, report.thresholds.medium_cpu_percent, report.thresholds.medium_io_mb_s, report.skipped_processes)}</p>
        {report.warnings?.map((warning) => <InlineAlert key={warning} message={warning} />)}
      </> : <section className="panel background-intro"><Gauge size={30} /><EmptyState title={c.empty} detail={c.emptyDetail} /></section>}
    </div>
  )
}

function ProcessRow({ process, onSection }: { process: BackgroundProcess; onSection: (section: TargetSection) => void }) {
  const { language } = useLanguage()
  const copy = backgroundCopy[language]
  const target = process.advice === 'review_startup' ? 'startup' : process.advice === 'review_service' ? 'services' : null
  return <tr><td title={process.executable}><span className="cell-main">{process.name}</span><span className="cell-sub">PID {process.pid}{process.executable ? ` · ${process.executable}` : ''}</span></td><td><div className="background-activity"><span className={`risk-badge risk-badge--${process.impact}`}>{copy.impacts[process.impact]}</span><strong>{process.cpu_percent.toFixed(2)}% CPU</strong><span>{copy.readWrite(process.read_mb_s, process.write_mb_s)}</span></div></td><td><span className="cell-main">{process.working_set_mb.toFixed(0)} MB</span><span className="cell-sub">{copy.threads(process.threads)}</span></td><td><div className="background-links">{process.startup.map((entry) => <span className="origin-badge" key={`startup-${entry.scope}-${entry.name}`}>{copy.startup}: {entry.name}</span>)}{process.services.map((service) => <span className={`origin-badge ${service.system ? 'origin-badge--system' : ''} ${service.critical ? 'origin-badge--critical' : ''}`} key={`service-${service.name}`}>{copy.service}: {service.display_name || service.name}</span>)}{!process.startup.length && !process.services.length ? <span className="cell-muted">{copy.none}</span> : null}</div></td><td><div className="advisor-copy"><strong>{copy.advice[process.advice].title}</strong><span>{copy.advice[process.advice].detail}</span>{target ? <Button variant="quiet" onClick={() => onSection(target)}>{target === 'startup' ? copy.openStartup : copy.openServices}<ArrowRight size={13} /></Button> : null}</div></td></tr>
}

const backgroundCopy = {
  en: {
    title: 'Background load advisor', description: 'Measure current process CPU and I/O activity, then correlate exact PIDs with startup entries and services. The advisor changes nothing by itself.', measuring: 'Measuring 1.5 s…', measure: 'Measure 1.5 s', measureAgain: 'Measure again', summary: 'Background-load measurement', sample: 'Actual sample', measured: 'Measured processes', cpu: 'Observed CPU', correlated: 'Correlated', filter: 'Process filter', active: 'Relevant activity', all: 'Top 64', io: (read: number, write: number) => `I/O deltas · read ${read.toFixed(2)} MB/s · write ${write.toFixed(2)} MB/s`, process: 'Process', activity: 'Measured activity', memory: 'Memory', links: 'Exact links', result: 'Advisor result', impacts: { low: 'Low', medium: 'Medium', high: 'High' }, readWrite: (read: number, write: number) => `R ${read.toFixed(2)} · W ${write.toFixed(2)} MB/s`, threads: (count: number) => `${count} threads`, startup: 'Startup', service: 'Service', none: 'No exact startup/service link', openStartup: 'Open Startup', openServices: 'Open Services', noActive: 'No relevant activity', noActiveDetail: 'Show Top 64 or measure again while the unwanted background workload is active.', empty: 'No measurement yet', emptyDetail: 'Run a bounded 1.5-second pass before a game. Repeat under the same idle conditions after changing one startup or service setting.', method: (cpus: number, cpu: number, io: number, skipped: number) => `CPU is normalized across ${cpus} logical processors and its threshold adapts to topology. Medium starts at ${cpu}% CPU or ${io} MB/s process I/O; memory is shown but never used alone to recommend a change. Skipped snapshot entries: ${skipped}.`, advice: { observe: { title: 'Observe only', detail: 'No reversible startup or service target was proven.' }, review_startup: { title: 'Review startup', detail: 'This active process matches an enabled current-user startup entry.' }, review_service: { title: 'Review service', detail: 'This activity maps to a manageable third-party service.' }, protected_service: { title: 'Protected service', detail: 'System, critical or read-only service: no disable recommendation.' } },
  },
  ru: {
    title: 'Советник фоновой нагрузки', description: 'Измеряет текущую нагрузку процессов на CPU и I/O, затем связывает точные PID с автозагрузкой и службами. Сам советник ничего не изменяет.', measuring: 'Измерение 1,5 с…', measure: 'Измерить 1,5 с', measureAgain: 'Измерить снова', summary: 'Замер фоновой нагрузки', sample: 'Реальный интервал', measured: 'Измерено процессов', cpu: 'Нагрузка CPU', correlated: 'Есть связь', filter: 'Фильтр процессов', active: 'Заметная активность', all: 'Топ-64', io: (read: number, write: number) => `Активность I/O · чтение ${read.toFixed(2)} MB/s · запись ${write.toFixed(2)} MB/s`, process: 'Процесс', activity: 'Измеренная активность', memory: 'Память', links: 'Точные связи', result: 'Результат советника', impacts: { low: 'Низкая', medium: 'Средняя', high: 'Высокая' }, readWrite: (read: number, write: number) => `R ${read.toFixed(2)} · W ${write.toFixed(2)} MB/s`, threads: (count: number) => `Потоков: ${count}`, startup: 'Автозагрузка', service: 'Служба', none: 'Точная связь не найдена', openStartup: 'Открыть автозагрузку', openServices: 'Открыть службы', noActive: 'Заметной активности нет', noActiveDetail: 'Откройте топ-64 или повторите замер во время нежелательной фоновой нагрузки.', empty: 'Замера ещё нет', emptyDetail: 'Запустите ограниченный замер на 1,5 секунды перед игрой. После одного изменения повторите его в таких же условиях простоя.', method: (cpus: number, cpu: number, io: number, skipped: number) => `CPU нормализован по ${cpus} логическим процессорам, а порог адаптируется к их числу. Средняя нагрузка начинается с ${cpu}% CPU или ${io} MB/s process I/O; память показывается, но сама по себе не приводит к совету. Пропущено snapshot-записей: ${skipped}.`, advice: { observe: { title: 'Только наблюдение', detail: 'Обратимая цель автозагрузки или службы не подтверждена.' }, review_startup: { title: 'Проверить автозагрузку', detail: 'Активный процесс совпадает с включённой пользовательской автозагрузкой.' }, review_service: { title: 'Проверить службу', detail: 'Активность связана с управляемой сторонней службой.' }, protected_service: { title: 'Защищённая служба', detail: 'Системная, критическая или read-only служба: совет отключения не выдаётся.' } },
  },
} as const
