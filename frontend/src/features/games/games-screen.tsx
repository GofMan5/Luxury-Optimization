import { useCallback, useMemo, useState, type FormEvent } from 'react'
import { Gamepad2, History, Play, Radar, Save, Trash2 } from 'lucide-react'
import type { RouteID } from '../../app/routes'
import { useBackend } from '../../app/backend-context'
import { useLanguage } from '../../app/language-context'
import type { GameHistoryReport, GameInstall, GameLaunchResult, GamesReport, SavedGame, SavedGames } from '../../shared/contracts/domain'
import { useBackendResource } from '../../shared/hooks/use-backend-resource'
import { Button } from '../../shared/ui/button'
import { ConfirmDialog } from '../../shared/ui/confirm-dialog'
import { EmptyState, InlineAlert, LoadingState } from '../../shared/ui/feedback'
import { PageHeader } from '../../shared/ui/page-header'

interface GameForm {
  path: string
  name: string
  profile: 'lite' | 'medium' | 'max'
  priority: 'normal' | 'above-normal' | 'high'
  affinity: string
  args: string
}

const initialForm: GameForm = { path: '', name: '', profile: 'medium', priority: 'normal', affinity: '', args: '' }

export default function GamesScreen({ onNavigate }: { onNavigate: (route: RouteID) => void }) {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = gamesCopy[language]
  const load = useCallback((signal: AbortSignal) => client.call<SavedGames>('gaming.saved', {}, signal), [client])
  const resource = useBackendResource(load, [load])
  const [form, setForm] = useState<GameForm>(initialForm)
  const [scan, setScan] = useState<GamesReport | null>(null)
  const [selected, setSelected] = useState<string | null>(null)
  const [remove, setRemove] = useState<SavedGame | null>(null)
  const [historyRevision, setHistoryRevision] = useState(0)
  const [busy, setBusy] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const saved = resource.data?.games ?? []
  const selectedGame = saved.find((game) => game.id === selected) ?? null
  const discovered = useMemo(() => flattenDiscovered(scan?.games ?? []), [scan])

  const scanLibraries = async () => {
    setBusy('scan'); setError(null)
    try { setScan(await client.call<GamesReport>('gaming.scan', {})) }
    catch (reason) { setError(messageOf(reason)) }
    finally { setBusy(null) }
  }

  const saveGame = async (request: { path: string; name?: string; profile?: string; priority?: string; affinity?: string; args?: string[] }) => {
    setBusy(`save:${request.path}`); setError(null)
    try {
      const game = await client.call<SavedGame>('gaming.save', request)
      setNotice(c.saved(game.name)); setSelected(game.id); setHistoryRevision((value) => value + 1)
      client.invalidate('gaming.'); resource.refresh()
      return true
    } catch (reason) { setError(messageOf(reason)); return false }
    finally { setBusy(null) }
  }

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    const success = await saveGame({ path: form.path, name: form.name.trim(), profile: form.profile, priority: form.priority, affinity: form.affinity.trim(), args: form.args.split(/\r?\n/).map((value) => value.trim()).filter(Boolean) })
    if (success) setForm((value) => ({ ...initialForm, profile: value.profile, priority: value.priority }))
  }

  const launch = async (game: SavedGame) => {
    setBusy(`launch:${game.id}`); setError(null)
    try {
      const result = await client.call<GameLaunchResult>('gaming.launch', { id: game.id })
      setNotice(c.launched(result.name, result.pid)); if (result.warning) setError(result.warning)
      setSelected(game.id); setHistoryRevision((value) => value + 1)
    } catch (reason) { setError(messageOf(reason)) }
    finally { setBusy(null) }
  }

  const removeGame = async () => {
    if (!remove) return
    setBusy(`remove:${remove.id}`); setError(null)
    try {
      await client.call('gaming.remove', { id: remove.id })
      setNotice(c.removed(remove.name)); if (selected === remove.id) setSelected(null)
      setRemove(null); client.invalidate('gaming.'); resource.refresh()
    } catch (reason) { setError(messageOf(reason)) }
    finally { setBusy(null) }
  }

  if (resource.loading && !resource.data) return <div className="page"><LoadingState label={c.loading} /></div>
  return (
    <div className="page">
      <PageHeader title={c.title} description={c.description} actions={<Button variant="primary" disabled={busy !== null} onClick={() => void scanLibraries()}><Radar size={16} />{busy === 'scan' ? c.scanning : c.scan}</Button>} />
      {resource.error ? <InlineAlert message={resource.error} onRetry={() => { client.invalidate('gaming.'); resource.refresh() }} /> : null}
      {error ? <InlineAlert message={error} /> : null}
      {notice ? <div className="success-notice" role="status">{notice}<button onClick={() => setNotice(null)} aria-label={c.dismiss}>×</button></div> : null}

      <section className="system-summary game-summary" aria-label={c.summary}>
        <div><span>{c.savedCount}</span><strong>{saved.length}</strong></div>
        <div><span>{c.detectedCount}</span><strong>{scan?.games.length ?? '—'}</strong></div>
        <div><span>{c.profileDefault}</span><strong>Medium</strong></div>
        <div><span>{c.evidence}</span><strong>{selectedGame ? c.open : c.select}</strong></div>
      </section>

      <div className="games-layout">
        <section className="panel game-table-panel">
          <div className="panel__header"><div><h2>{c.savedProfiles}</h2><p>{c.savedDescription}</p></div></div>
          {saved.length ? <table className="data-table game-table"><thead><tr><th>{c.game}</th><th>{c.profile}</th><th>{c.path}</th><th aria-label={c.actions} /></tr></thead><tbody>{saved.map((game) => <tr key={game.id}><td><span className="cell-main">{game.name}</span><span className="cell-sub">{game.priority}{game.affinity ? ` · ${c.customAffinity}` : ''}</span></td><td><span className={`risk-badge risk-badge--${profileRisk(game.profile)}`}>{game.profile}</span></td><td className="path-cell" title={game.path}>{game.path}</td><td><div className="table-actions game-actions"><Button variant="quiet" onClick={() => { setSelected(game.id); setHistoryRevision((value) => value + 1) }}><History size={14} />{c.history}</Button><Button variant="secondary" disabled={busy !== null} onClick={() => void launch(game)}><Play size={14} />{busy === `launch:${game.id}` ? c.launching : c.launch}</Button><Button variant="quiet" disabled={busy !== null} aria-label={c.remove(game.name)} title={c.remove(game.name)} onClick={() => setRemove(game)}><Trash2 size={14} /></Button></div></td></tr>)}</tbody></table> : <EmptyState title={c.emptySaved} detail={c.emptySavedDetail} />}
        </section>

        <section className="panel game-form-panel">
          <div className="panel__header"><div><h2>{c.manual}</h2><p>{c.manualDescription}</p></div></div>
          <form className="panel__body game-form" onSubmit={(event) => void submit(event)}>
            <div className="field"><label htmlFor="game-path">{c.executable}</label><input id="game-path" value={form.path} required spellCheck="false" onChange={(event) => setForm({ ...form, path: event.target.value })} placeholder={c.pathPlaceholder} /></div>
            <div className="field"><label htmlFor="game-name">{c.name}</label><input id="game-name" value={form.name} maxLength={128} onChange={(event) => setForm({ ...form, name: event.target.value })} placeholder={c.namePlaceholder} /></div>
            <div className="form-grid"><div className="field"><label htmlFor="game-profile">{c.profile}</label><select id="game-profile" value={form.profile} onChange={(event) => setForm({ ...form, profile: event.target.value as GameForm['profile'] })}><option value="lite">Lite</option><option value="medium">Medium</option><option value="max">Max</option></select></div><div className="field"><label htmlFor="game-priority">{c.priority}</label><select id="game-priority" value={form.priority} onChange={(event) => setForm({ ...form, priority: event.target.value as GameForm['priority'] })}><option value="normal">Normal</option><option value="above-normal">Above normal</option><option value="high">High</option></select></div></div>
            <div className="field"><label htmlFor="game-affinity">{c.affinity}</label><input id="game-affinity" value={form.affinity} onChange={(event) => setForm({ ...form, affinity: event.target.value })} placeholder="0xFF" /></div>
            <div className="field"><label htmlFor="game-args">{c.arguments}</label><textarea id="game-args" value={form.args} onChange={(event) => setForm({ ...form, args: event.target.value })} placeholder={c.argumentsPlaceholder} /></div>
            <Button variant="primary" type="submit" disabled={busy !== null}><Save size={15} />{c.save}</Button>
          </form>
        </section>
      </div>

      {selectedGame ? <GameHistoryPanel key={`${selectedGame.id}-${historyRevision}`} game={selectedGame} onBenchmarks={() => onNavigate('benchmarks')} /> : null}

      {scan ? <section className="panel discovered-games"><div className="panel__header"><div><h2>{c.discovered}</h2><p>{c.discoveredDescription}</p></div></div>{discovered.length ? <table className="data-table"><thead><tr><th>{c.game}</th><th>{c.source}</th><th>{c.executable}</th><th aria-label={c.actions} /></tr></thead><tbody>{discovered.map((item) => <tr key={`${item.game.source}-${item.game.id}-${item.path}`}><td><span className="cell-main">{item.game.name}</span></td><td>{item.game.source}</td><td className="path-cell" title={item.path}>{item.path}</td><td><div className="table-actions"><Button variant="quiet" disabled={busy !== null} onClick={() => void saveGame({ path: item.path, name: item.game.name, profile: 'medium', priority: 'normal' })}><Save size={14} />{c.saveMedium}</Button></div></td></tr>)}</tbody></table> : <EmptyState title={c.emptyDetected} detail={c.emptyDetectedDetail} />}{scan.warnings?.map((warning) => <InlineAlert key={warning} message={warning} />)}</section> : null}

      <ConfirmDialog open={remove !== null} title={c.removeTitle(remove?.name ?? '')} description={<><p>{c.removeDescription}</p><ul><li>{c.removeRule1}</li><li>{c.removeRule2}</li></ul></>} confirmLabel={c.removeConfirm} danger busy={busy?.startsWith('remove:')} onCancel={() => setRemove(null)} onConfirm={() => void removeGame()} />
    </div>
  )
}

function GameHistoryPanel({ game, onBenchmarks }: { game: SavedGame; onBenchmarks: () => void }) {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = gamesCopy[language]
  const load = useCallback((signal: AbortSignal) => client.call<GameHistoryReport>('gaming.history', { id: game.id }, signal), [client, game.id])
  const resource = useBackendResource(load, [load])
  if (resource.loading && !resource.data) return <LoadingState label={c.loadingHistory} />
  return <section className="panel game-history-panel"><div className="panel__header"><div><h2>{c.historyFor(game.name)}</h2><p>{c.historyDescription}</p></div><Button variant="secondary" onClick={onBenchmarks}><Gamepad2 size={15} />{c.openBenchmarks}</Button></div>{resource.error ? <InlineAlert message={resource.error} onRetry={resource.refresh} /> : null}<div className="panel__body game-history-body"><div><h3>{c.attachedResults}</h3>{resource.data?.benchmarks.length ? <div className="game-results">{resource.data.benchmarks.map((item) => <article className="game-result" key={item.id}><div><strong className={verdictTone(item.comparison.verdict)}>{c.verdicts[item.comparison.verdict]}</strong><time>{formatTime(item.created_at, language)}</time></div><dl><div><dt>{c.average}</dt><dd>{signed(item.comparison.average_fps.delta_percent)}%</dd></div><div><dt>{c.low}</dt><dd>{signed(item.comparison.one_percent_low_fps.delta_percent)}%</dd></div><div><dt>{c.frame}</dt><dd>{signed(item.comparison.p95_frame_ms.delta_percent)}%</dd></div></dl></article>)}</div> : <EmptyState title={c.noResults} detail={c.noResultsDetail} />}</div><div><h3>{c.recentLaunches}</h3>{resource.data?.launches.length ? <table className="data-table compact-table"><thead><tr><th>{c.started}</th><th>{c.profile}</th><th>{c.launcherPID}</th></tr></thead><tbody>{resource.data.launches.map((launch) => <tr key={launch.id}><td><span className="cell-main">{formatTime(launch.started_at, language)}</span></td><td>{launch.profile} · {launch.priority}</td><td>#{launch.launcher_pid}</td></tr>)}</tbody></table> : <EmptyState title={c.noLaunches} detail={c.noLaunchesDetail} />}</div></div></section>
}

function flattenDiscovered(games: GameInstall[]): Array<{ game: GameInstall; path: string }> {
  return games.flatMap((game) => (game.executables ?? []).map((path) => ({ game, path })))
}

function profileRisk(profile: SavedGame['profile']): 'low' | 'medium' | 'high' {
  return profile === 'lite' || profile === 'recommended' ? 'low' : profile === 'medium' ? 'medium' : 'high'
}

function verdictTone(verdict: keyof typeof gamesCopy.en.verdicts): string {
  return verdict === 'measurably_improved' ? 'text-success' : verdict === 'measurably_regressed' ? 'text-warning' : ''
}

function formatTime(value: string, language: 'en' | 'ru'): string {
  return new Date(value).toLocaleString(language === 'ru' ? 'ru-RU' : 'en-US')
}

function signed(value: number): string {
  return `${value > 0 ? '+' : ''}${value.toFixed(2)}`
}

function messageOf(value: unknown): string {
  return value instanceof Error ? value.message : String(value)
}

const gamesCopy = {
  en: {
    loading: 'Loading saved game profiles…', title: 'Games', description: 'Save exact launch settings, run games through a bounded session boost, and keep measured evidence attached to each game.', scan: 'Scan libraries', scanning: 'Scanning…', dismiss: 'Dismiss notification', summary: 'Game profile summary', savedCount: 'Saved profiles', detectedCount: 'Detected games', profileDefault: 'Discovery default', evidence: 'Evidence panel', open: 'Open', select: 'Select game', savedProfiles: 'Saved launch profiles', savedDescription: 'A launch never elevates the game; the helper restores session changes after exit.', game: 'Game', profile: 'Profile', path: 'Executable', actions: 'Actions', customAffinity: 'custom affinity', history: 'History', launch: 'Launch', launching: 'Starting…', remove: (name: string) => `Remove ${name}`, emptySaved: 'No saved game profiles', emptySavedDetail: 'Scan Steam/Epic/Xbox or save one executable manually.', manual: 'Manual profile', manualDescription: 'One executable, one explicit profile. Enter one argument per line.', executable: 'Executable path', name: 'Display name', priority: 'Priority', affinity: 'Optional CPU affinity', arguments: 'Arguments', pathPlaceholder: 'C:\\Games\\Game\\game.exe', namePlaceholder: 'Derived from executable when empty', argumentsPlaceholder: '-windowed\n-novid', save: 'Save profile', saved: (name: string) => `${name}: launch profile saved.`, launched: (name: string, pid: number) => `${name}: bounded launch started (helper PID ${pid}).`, removed: (name: string) => `${name}: profile removed.`, discovered: 'Discovered executables', discoveredDescription: 'Read-only manifests and bounded executable scanning. Save uses Medium + normal priority by default.', source: 'Source', saveMedium: 'Save Medium', emptyDetected: 'No supported executables found', emptyDetectedDetail: 'The scan reports only executables that pass platform filters.', removeTitle: (name: string) => `Remove ${name}?`, removeDescription: 'This removes only the saved launch profile.', removeRule1: 'No game files or benchmark evidence are deleted', removeRule2: 'The optimizer configuration for other games is unchanged', removeConfirm: 'Remove profile', loadingHistory: 'Loading launch evidence…', historyFor: (name: string) => `${name} evidence`, historyDescription: 'Recent bounded launches and explicit before/after benchmark attachments.', openBenchmarks: 'Open Benchmarks', attachedResults: 'Attached benchmark results', recentLaunches: 'Recent launches', noResults: 'No attached measurements', noResultsDetail: 'Compare runs in Benchmarks and attach the verdict to this game.', noLaunches: 'No recorded launches', noLaunchesDetail: 'Launch this saved profile to create the first record.', started: 'Started', launcherPID: 'Helper PID', average: 'Average FPS', low: '1% low', frame: 'p95 frametime', verdicts: { measurably_improved: 'Measurably improved', measurably_regressed: 'Measurably regressed', mixed_result: 'Mixed result', within_run_to_run_variance: 'Inside variance' },
  },
  ru: {
    loading: 'Загрузка сохранённых игровых профилей…', title: 'Игры', description: 'Сохраняйте точные параметры запуска, запускайте игры через ограниченный сессионный буст и прикрепляйте измеримые результаты к каждой игре.', scan: 'Найти игры', scanning: 'Поиск…', dismiss: 'Закрыть уведомление', summary: 'Сводка игровых профилей', savedCount: 'Сохранено', detectedCount: 'Найдено игр', profileDefault: 'Профиль поиска', evidence: 'Панель замеров', open: 'Открыта', select: 'Выберите игру', savedProfiles: 'Сохранённые профили запуска', savedDescription: 'Игра не получает elevation; после выхода помощник возвращает сессионные изменения.', game: 'Игра', profile: 'Профиль', path: 'Исполняемый файл', actions: 'Действия', customAffinity: 'своя affinity', history: 'История', launch: 'Запустить', launching: 'Запуск…', remove: (name: string) => `Удалить ${name}`, emptySaved: 'Сохранённых профилей нет', emptySavedDetail: 'Найдите Steam/Epic/Xbox или сохраните исполняемый файл вручную.', manual: 'Ручной профиль', manualDescription: 'Один исполняемый файл и один явный профиль. Каждый аргумент вводится с новой строки.', executable: 'Путь к игре', name: 'Название', priority: 'Приоритет', affinity: 'Необязательная CPU affinity', arguments: 'Аргументы', pathPlaceholder: 'C:\\Games\\Game\\game.exe', namePlaceholder: 'Пустое поле возьмёт имя файла', argumentsPlaceholder: '-windowed\n-novid', save: 'Сохранить профиль', saved: (name: string) => `${name}: профиль запуска сохранён.`, launched: (name: string, pid: number) => `${name}: ограниченный запуск начат (PID помощника ${pid}).`, removed: (name: string) => `${name}: профиль удалён.`, discovered: 'Найденные исполняемые файлы', discoveredDescription: 'Только чтение manifest и ограниченный поиск. По умолчанию сохраняются Medium и normal priority.', source: 'Источник', saveMedium: 'Сохранить Medium', emptyDetected: 'Поддерживаемых файлов не найдено', emptyDetectedDetail: 'Показываются только исполняемые файлы, прошедшие платформенные фильтры.', removeTitle: (name: string) => `Удалить ${name}?`, removeDescription: 'Удаляется только сохранённый профиль запуска.', removeRule1: 'Файлы игры и история замеров сохраняются', removeRule2: 'Настройки других игр не меняются', removeConfirm: 'Удалить профиль', loadingHistory: 'Загрузка истории запусков…', historyFor: (name: string) => `История и замеры: ${name}`, historyDescription: 'Последние ограниченные запуски и явно прикреплённые сравнения до/после.', openBenchmarks: 'Открыть замеры', attachedResults: 'Прикреплённые результаты', recentLaunches: 'Последние запуски', noResults: 'Прикреплённых замеров нет', noResultsDetail: 'Сравните прогоны в разделе «Замеры» и прикрепите итог к этой игре.', noLaunches: 'Запусков ещё нет', noLaunchesDetail: 'Запустите сохранённый профиль, чтобы создать первую запись.', started: 'Запущено', launcherPID: 'PID помощника', average: 'Средний FPS', low: '1% low', frame: 'p95 frametime', verdicts: { measurably_improved: 'Измеримо лучше', measurably_regressed: 'Измеримо хуже', mixed_result: 'Смешанный результат', within_run_to_run_variance: 'В пределах разброса' },
  },
} as const
