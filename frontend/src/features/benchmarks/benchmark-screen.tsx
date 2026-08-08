import { useCallback, useRef, useState, type ChangeEvent, type FormEvent } from 'react'
import { BarChart3, Link2, Play, Upload } from 'lucide-react'
import { useBackend } from '../../app/backend-context'
import { useLanguage } from '../../app/language-context'
import type { BenchmarkComparison, BenchmarkRun, BenchmarkSet, GameBenchmarkAttachment, MetricComparison, SavedGames } from '../../shared/contracts/domain'
import { useBackendResource } from '../../shared/hooks/use-backend-resource'
import { Button } from '../../shared/ui/button'
import { InlineAlert } from '../../shared/ui/feedback'
import { PageHeader } from '../../shared/ui/page-header'
import { importBenchmarkFiles } from './benchmark-import'

const emptyRuns = (): BenchmarkRun[] => Array.from({ length: 3 }, () => ({ average_fps: 0, one_percent_low_fps: 0, p95_frame_ms: 0 }))

interface BenchmarkCopy {
  title: string; description: string; before: string; after: string; beforeTitle: string; afterTitle: string; compare: string; comparing: string; import: string; importing: string; run: string; average: string; low: string; frame: string; validation: string; method: string; verdict: string; noise: string; attachTitle: string; attachDescription: string; selectGame: string; attach: string; attaching: string; noGames: string; attached: (name: string) => string
  runCount: (count: number) => string
  verdicts: Record<BenchmarkComparison['verdict'], string>
}

export default function BenchmarkScreen({ embedded = false }: { embedded?: boolean }) {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = benchmarkCopy[language]
  const loadGames = useCallback((signal: AbortSignal) => client.call<SavedGames>('gaming.saved', {}, signal), [client])
  const games = useBackendResource(loadGames, [loadGames])
  const [before, setBefore] = useState(emptyRuns)
  const [after, setAfter] = useState(emptyRuns)
  const [importing, setImporting] = useState<'before' | 'after' | null>(null)
  const [comparing, setComparing] = useState(false)
  const [comparison, setComparison] = useState<BenchmarkComparison | null>(null)
  const [compared, setCompared] = useState<{ before: BenchmarkSet; after: BenchmarkSet } | null>(null)
  const [gameID, setGameID] = useState('')
  const [attaching, setAttaching] = useState(false)
  const [attached, setAttached] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const importSide = async (side: 'before' | 'after', event: ChangeEvent<HTMLInputElement>) => {
    const input = event.currentTarget
    const files = Array.from(input.files ?? [])
    if (importing || files.length === 0) return
    setImporting(side); setError(null); setComparison(null); setCompared(null); setAttached(null)
    try {
      const runs = await importBenchmarkFiles(files)
      if (side === 'before') setBefore(runs); else setAfter(runs)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      input.value = ''
      setImporting(null)
    }
  }

  const compare = async (event: FormEvent) => {
    event.preventDefault()
    if (importing || !validRuns(before) || !validRuns(after)) { setError(c.validation); return }
    setComparing(true); setError(null)
    const payload: { before: BenchmarkSet; after: BenchmarkSet } = { before: { label: c.before, runs: before }, after: { label: c.after, runs: after } }
    try { setComparison(await client.call<BenchmarkComparison>('benchmark.compare', payload)); setCompared(payload); setAttached(null) }
    catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)) }
    finally { setComparing(false) }
  }

  const attach = async () => {
    const resolvedGameID = gameID || games.data?.games[0]?.id || ''
    if (!compared || !resolvedGameID) return
    setAttaching(true); setError(null)
    try {
      await client.call<GameBenchmarkAttachment>('gaming.attach_benchmark', { game_id: resolvedGameID, ...compared })
      const game = games.data?.games.find((item) => item.id === resolvedGameID)
      setAttached(c.attached(game?.name ?? resolvedGameID))
    } catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)) }
    finally { setAttaching(false) }
  }

  return (
    <div className={embedded ? 'system-pane' : 'page'}>
      {embedded ? <div className="system-pane-heading"><div><h2>{c.title}</h2><p>{c.description}</p></div><Button variant="primary" form="benchmark-form" type="submit" disabled={comparing || importing !== null}><Play size={16} />{comparing ? c.comparing : c.compare}</Button></div> : <PageHeader title={c.title} description={c.description} actions={<Button variant="primary" form="benchmark-form" type="submit" disabled={comparing || importing !== null}><Play size={16} />{comparing ? c.comparing : c.compare}</Button>} />}
      {error ? <InlineAlert message={error} /> : null}
      <form id="benchmark-form" onSubmit={(event) => void compare(event)}>
        <div className="benchmark-layout">
          <RunPanel title={c.beforeTitle} runs={before} onRuns={(runs) => { setBefore(runs); setComparison(null); setCompared(null); setAttached(null); setError(null) }} disabled={importing !== null || comparing} importing={importing === 'before'} onImport={(event) => void importSide('before', event)} copy={c} />
          <RunPanel title={c.afterTitle} runs={after} onRuns={(runs) => { setAfter(runs); setComparison(null); setCompared(null); setAttached(null); setError(null) }} disabled={importing !== null || comparing} importing={importing === 'after'} onImport={(event) => void importSide('after', event)} copy={c} />
        </div>
      </form>
      <p className="method-note">{c.method}</p>
      {comparison ? <><Result comparison={comparison} copy={c} /><section className="panel benchmark-attachment"><div><h2>{c.attachTitle}</h2><p>{c.attachDescription}</p></div>{games.error ? <InlineAlert message={games.error} onRetry={games.refresh} /> : null}{games.data?.games.length ? <div className="benchmark-attachment__actions"><select aria-label={c.selectGame} value={gameID || games.data.games[0]?.id} disabled={attaching} onChange={(event) => { setGameID(event.target.value); setAttached(null) }}>{games.data.games.map((game) => <option key={game.id} value={game.id}>{game.name}</option>)}</select><Button variant="secondary" disabled={attaching || attached !== null} onClick={() => void attach()}><Link2 size={15} />{attaching ? c.attaching : c.attach}</Button></div> : games.error ? null : <span className="cell-muted">{c.noGames}</span>}{attached ? <strong className="text-success">{attached}</strong> : null}</section></> : null}
    </div>
  )
}

function RunPanel({ title, runs, onRuns, disabled, importing, onImport, copy }: { title: string; runs: BenchmarkRun[]; onRuns: (runs: BenchmarkRun[]) => void; disabled: boolean; importing: boolean; onImport: (event: ChangeEvent<HTMLInputElement>) => void; copy: BenchmarkCopy }) {
  const fileInput = useRef<HTMLInputElement>(null)
  const update = (index: number, key: keyof BenchmarkRun, value: string) => onRuns(runs.map((run, runIndex) => runIndex === index ? { ...run, [key]: Number(value) } : run))
  return (
    <section className="panel benchmark-set">
      <div className="panel__header"><div><h2>{title}</h2><p>{copy.runCount(runs.length)}</p></div><div className="benchmark-import"><Button type="button" variant="secondary" disabled={disabled} onClick={() => fileInput.current?.click()}><Upload size={15} />{importing ? copy.importing : copy.import}</Button><input ref={fileInput} aria-label={copy.import} type="file" accept=".json,.csv,application/json,text/csv" multiple hidden onChange={onImport} /></div></div>
      <div className="panel__body">
        <div className="benchmark-grid benchmark-grid--header"><span>{copy.run}</span><span>{copy.average}</span><span>{copy.low}</span><span>{copy.frame}</span></div>
        {runs.map((run, index) => <div className="benchmark-grid" key={index}><strong>{index + 1}</strong><MetricInput label={`${title} ${index + 1} ${copy.average}`} value={run.average_fps} disabled={disabled} onChange={(value) => update(index, 'average_fps', value)} /><MetricInput label={`${title} ${index + 1} ${copy.low}`} value={run.one_percent_low_fps} disabled={disabled} onChange={(value) => update(index, 'one_percent_low_fps', value)} /><MetricInput label={`${title} ${index + 1} ${copy.frame}`} value={run.p95_frame_ms} disabled={disabled} onChange={(value) => update(index, 'p95_frame_ms', value)} /></div>)}
      </div>
    </section>
  )
}

function MetricInput({ label, value, disabled, onChange }: { label: string; value: number; disabled: boolean; onChange: (value: string) => void }) {
  return <input aria-label={label} type="number" min="0.001" max="100000" step="0.001" value={value || ''} required disabled={disabled} onChange={(event) => onChange(event.target.value)} />
}

function Result({ comparison, copy }: { comparison: BenchmarkComparison; copy: BenchmarkCopy }) {
  const tone = comparison.verdict === 'measurably_improved' ? 'text-success' : comparison.verdict === 'measurably_regressed' ? 'text-warning' : ''
  return (
    <section className="panel panel--gold benchmark-result" aria-live="polite">
      <div className="benchmark-verdict"><div><span>{copy.verdict}</span><h2 className={tone}>{copy.verdicts[comparison.verdict]}</h2></div><BarChart3 size={30} aria-hidden="true" /></div>
      <div className="metric-grid"><ComparisonMetric label={copy.average} metric={comparison.average_fps} suffix=" FPS" noise={copy.noise} /><ComparisonMetric label={copy.low} metric={comparison.one_percent_low_fps} suffix=" FPS" noise={copy.noise} /><ComparisonMetric label={copy.frame} metric={comparison.p95_frame_ms} suffix=" ms" noise={copy.noise} /></div>
    </section>
  )
}

function ComparisonMetric({ label, metric, suffix, noise }: { label: string; metric: MetricComparison; suffix: string; noise: string }) {
  const sign = metric.delta_percent > 0 ? '+' : ''
  return <div className="metric"><span className="metric__label">{label}</span><strong className="metric__value">{metric.before_median.toFixed(2)} → {metric.after_median.toFixed(2)}{suffix}</strong><span className="metric__detail">{sign}{metric.delta_percent.toFixed(2)}% · {noise} {metric.noise_percent.toFixed(2)}%</span></div>
}

function validRuns(runs: BenchmarkRun[]): boolean {
  return runs.length >= 3 && runs.length <= 100 && runs.every((run) => Object.values(run).every((value) => value > 0 && value <= 100_000 && Number.isFinite(value)))
}

const benchmarkCopy: Record<'en' | 'ru', BenchmarkCopy> = {
  en: {
    title: 'Benchmarks', description: 'Prove whether a tweak helped: compare at least three identical before/after passes and reject changes inside normal run-to-run variance.', before: 'Before', after: 'After', beforeTitle: 'Baseline runs', afterTitle: 'Optimized runs', compare: 'Compare runs', comparing: 'Comparing…', import: 'Import captures', importing: 'Importing…', run: 'Run', average: 'Average FPS', low: '1% low FPS', frame: 'p95 frametime', noise: 'noise', runCount: (count: number) => `${count} runs · manual or imported`, validation: 'Enter or import 3–100 positive runs on each side.', method: 'Imports native Luxury JSON, CapFrameX JSON (MsBetweenPresents), and raw MangoHud CSV logs. Captures are normalized to average FPS, percentile 1% low, and p95 frametime; final significance uses median and MAD noise.', verdict: 'Measured verdict', attachTitle: 'Attach measured result', attachDescription: 'Keep this exact before/after evidence with one saved game profile.', selectGame: 'Saved game', attach: 'Attach to game', attaching: 'Attaching…', noGames: 'Save a game profile first.', attached: (name: string) => `Attached to ${name}.`, verdicts: { measurably_improved: 'Measurably improved', measurably_regressed: 'Measurably regressed', mixed_result: 'Mixed result', within_run_to_run_variance: 'Inside run-to-run variance' },
  },
  ru: {
    title: 'Бенчмарки', description: 'Проверьте реальную пользу твика: сравните минимум три одинаковых прогона до и после и отбросьте разницу внутри обычного разброса.', before: 'До', after: 'После', beforeTitle: 'Исходные прогоны', afterTitle: 'Прогоны после оптимизации', compare: 'Сравнить прогоны', comparing: 'Сравнение…', import: 'Импорт замеров', importing: 'Импорт…', run: 'Прогон', average: 'Средний FPS', low: '1% low FPS', frame: 'p95 frametime', noise: 'шум', runCount: (count: number) => `${count} прогонов · вручную или импорт`, validation: 'Введите или импортируйте 3–100 положительных прогонов с каждой стороны.', method: 'Поддерживаются Luxury JSON, CapFrameX JSON (MsBetweenPresents) и сырые логи MangoHud CSV. Замеры преобразуются в средний FPS, процентильный 1% low и p95 frametime; значимость определяется по медиане и шуму MAD.', verdict: 'Измеримый итог', attachTitle: 'Прикрепить измеримый результат', attachDescription: 'Сохраните это точное сравнение до/после у одного игрового профиля.', selectGame: 'Сохранённая игра', attach: 'Прикрепить к игре', attaching: 'Сохранение…', noGames: 'Сначала сохраните игровой профиль.', attached: (name: string) => `Результат прикреплён к ${name}.`, verdicts: { measurably_improved: 'Измеримо лучше', measurably_regressed: 'Измеримо хуже', mixed_result: 'Смешанный результат', within_run_to_run_variance: 'В пределах разброса прогонов' },
  },
}
