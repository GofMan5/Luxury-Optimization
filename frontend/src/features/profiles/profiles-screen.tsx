import { useCallback, useMemo, useState } from 'react'
import { Check, ChevronDown, Gauge, Search, ShieldCheck, SlidersHorizontal, Zap } from 'lucide-react'
import { useBackend } from '../../app/backend-context'
import { useLanguage } from '../../app/language-context'
import type { CheckpointStatus, MutationResult, Plan, PlanItem } from '../../shared/contracts/domain'
import { useBackendResource } from '../../shared/hooks/use-backend-resource'
import { Button } from '../../shared/ui/button'
import { ConfirmDialog } from '../../shared/ui/confirm-dialog'
import { InlineAlert, LoadingState } from '../../shared/ui/feedback'
import { PageHeader } from '../../shared/ui/page-header'
import { StatusDot } from '../../shared/ui/status'

type ProfileID = 'lite' | 'medium' | 'max'
type TweakSection = 'tweaks' | 'profiles'
type RiskFilter = 'all' | 'low' | 'medium' | 'high'
type SortMode = 'default' | 'name' | 'risk' | 'status'

interface TweakData {
  lite: Plan
  medium: Plan
  max: Plan
  checkpoint: CheckpointStatus
}

export default function ProfilesScreen() {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = tweaksCopy[language]
  const [section, setSection] = useState<TweakSection>('tweaks')
  const [profile, setProfile] = useState<ProfileID>('lite')
  const load = useCallback(async (signal: AbortSignal): Promise<TweakData> => {
    const [lite, medium, max, checkpoint] = await Promise.all([
      client.call<Plan>('optimization.plan', { profile: 'lite' }, signal),
      client.call<Plan>('optimization.plan', { profile: 'medium' }, signal),
      client.call<Plan>('optimization.plan', { profile: 'max' }, signal),
      client.call<CheckpointStatus>('optimization.checkpoint_status', { profile }, signal),
    ])
    return { lite, medium, max, checkpoint }
  }, [client, profile])
  const resource = useBackendResource(load, [load])
  const [confirming, setConfirming] = useState(false)
  const [applying, setApplying] = useState(false)
  const [checkpointing, setCheckpointing] = useState(false)
  const [result, setResult] = useState<string | null>(null)

  const selectedPlan = resource.data?.[profile] ?? null
  const changed = selectedPlan?.items.filter((item) => item.changed).length ?? 0

  const createCheckpoint = async () => {
    setCheckpointing(true); setResult(null)
    try {
      await client.call<CheckpointStatus>('optimization.create_checkpoint', { profile })
      setResult(c.checkpointCreated)
      resource.refresh()
    } catch (error) { setResult(error instanceof Error ? error.message : String(error)) }
    finally { setCheckpointing(false) }
  }

  const apply = async () => {
    setApplying(true); setResult(null)
    try {
      await client.call<MutationResult>('optimization.apply', { profile })
      setResult(c.applied)
      setConfirming(false)
      resource.refresh()
    } catch (error) { setResult(error instanceof Error ? error.message : String(error)) }
    finally { setApplying(false) }
  }

  if (resource.loading && !resource.data) return <div className="page"><LoadingState label={c.loading} /></div>
  return (
    <div className="page">
      <PageHeader title={c.title} description={c.description} />
      <div className="tweak-tabs" role="tablist" aria-label={c.title}>
        <button role="tab" aria-selected={section === 'tweaks'} onClick={() => setSection('tweaks')}><SlidersHorizontal size={17} />{c.tweaks}</button>
        <button role="tab" aria-selected={section === 'profiles'} onClick={() => setSection('profiles')}><Gauge size={17} />{c.profiles}</button>
      </div>
      {resource.error ? <InlineAlert message={resource.error} onRetry={() => { client.invalidate('optimization.'); resource.refresh() }} /> : null}
      {result ? <div className="success-notice" role="status"><ShieldCheck size={18} />{result}<button onClick={() => setResult(null)} aria-label={c.dismiss}>×</button></div> : null}
      {section === 'tweaks' && resource.data ? <TweakCatalog plan={resource.data.max} onChanged={(message) => { setResult(message); resource.refresh() }} /> : null}
      {section === 'profiles' && resource.data && selectedPlan ? (
        <>
          <div className="preset-grid">
            <PresetCard id="lite" selected={profile === 'lite'} title={c.lite} subtitle={c.liteSubtitle} description={c.liteDescription} count={resource.data.lite.items.length} risk={c.liteRisk} icon={<ShieldCheck size={22} />} onSelect={setProfile} />
            <PresetCard id="medium" selected={profile === 'medium'} title={c.mediumProfile} subtitle={c.mediumSubtitle} description={c.mediumDescription} count={resource.data.medium.items.length} risk={c.mediumRisk} icon={<Gauge size={22} />} onSelect={setProfile} />
            <PresetCard id="max" selected={profile === 'max'} title={c.max} subtitle={c.maxSubtitle} description={c.maxDescription} count={resource.data.max.items.length} risk={c.maxRisk} icon={<Zap size={22} />} onSelect={setProfile} />
          </div>
          <section className={`checkpoint-guard panel ${resource.data.checkpoint.ready ? 'checkpoint-guard--ready' : 'panel--gold'}`}>
            <div><ShieldCheck size={23} /><div><h2>{resource.data.checkpoint.ready ? c.checkpointReady : c.checkpointRequired}</h2><p>{resource.data.checkpoint.ready ? c.checkpointReadyDetail(resource.data.checkpoint.expires_at) : c.checkpointRequiredDetail}</p></div></div>
            {resource.data.checkpoint.ready ? <StatusDot tone="success" label={c.ready} /> : <Button variant="primary" disabled={checkpointing} onClick={() => void createCheckpoint()}>{checkpointing ? c.creating : c.createCheckpoint}</Button>}
          </section>
          <section className="preset-summary panel panel--gold"><div><span>{c.selectedPreset}</span><h2>{profile === 'lite' ? c.lite : profile === 'medium' ? c.mediumProfile : c.max}</h2><p>{profile === 'lite' ? c.liteIntent : profile === 'medium' ? c.mediumIntent : c.maxIntent}</p></div><div className="preset-summary__counts"><strong>{selectedPlan.items.length}</strong><span>{c.tweaksCount}</span><strong>{changed}</strong><span>{c.toChange}</span></div></section>
          <TweakList items={selectedPlan.items} />
          {selectedPlan.warnings?.map((warning) => <InlineAlert key={warning} message={warning} />)}
          <footer className="plan-footer panel"><div><strong>{resource.data.checkpoint.ready ? c.readyToApply : c.createFirst}</strong><span>{c.applyDetail}</span></div><Button variant="primary" disabled={changed === 0 || !resource.data.checkpoint.ready} onClick={() => setConfirming(true)}>{c.applyPreset(changed)}</Button></footer>
          <ConfirmDialog open={confirming} title={c.applyTitle(profile)} description={<><p>{c.transaction}</p><ul><li>{c.exactTargets(changed)}</li><li>{c.checkpointConfirmed}</li><li>{c.transactionalBackup}</li><li>{c.nativeReadback}</li></ul></>} confirmLabel={c.applyPreset(changed)} busy={applying} onCancel={() => setConfirming(false)} onConfirm={() => void apply()} />
        </>
      ) : null}
    </div>
  )
}

function TweakCatalog({ plan, onChanged }: { plan: Plan; onChanged: (message: string) => void }) {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = tweaksCopy[language]
  const [query, setQuery] = useState('')
  const [risk, setRisk] = useState<RiskFilter>('all')
  const [sort, setSort] = useState<SortMode>('default')
  const [busy, setBusy] = useState<{ id: string; action: 'apply' | 'restore' } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const items = useMemo(() => sortTweakItems(plan.items.filter((item) => {
    const text = `${item.name} ${item.category} ${item.effect} ${item.benefit}`.toLowerCase()
    return (risk === 'all' || item.risk_level === risk) && text.includes(query.toLowerCase())
  }), sort, language), [plan.items, query, risk, sort, language])

  const mutate = async (action: 'apply' | 'restore', item: PlanItem) => {
    setBusy({ id: item.id, action }); setError(null)
    try {
      const result = await client.call<MutationResult>(`optimization.${action}_tweak`, { id: item.id })
      onChanged(action === 'apply' ? (result.changed ? c.tweakApplied(localizePlan(item.name, language)) : c.noChange) : c.tweakRestored(localizePlan(item.name, language)))
    } catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)) }
    finally { setBusy(null) }
  }
  return (
    <>
      <section className="catalog-intro panel panel--gold"><div><h2>{c.allTweaks}</h2><p>{c.catalogDescription}</p></div><strong>{plan.items.length}</strong></section>
      <div className="toolbar tweak-toolbar"><label className="search-field"><Search size={16} /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={c.search} aria-label={c.search} /></label><label className="field tweak-sort"><span className="field__label">{c.sort}</span><select value={sort} onChange={(event) => setSort(event.target.value as SortMode)}><option value="default">{c.sortDefault}</option><option value="name">{c.sortName}</option><option value="risk">{c.sortRisk}</option><option value="status">{c.sortStatus}</option></select></label><div className="segmented" aria-label={c.riskFilter}><button aria-pressed={risk === 'all'} onClick={() => setRisk('all')}>{c.allRisks}</button><button aria-pressed={risk === 'low'} onClick={() => setRisk('low')}>{c.low}</button><button aria-pressed={risk === 'medium'} onClick={() => setRisk('medium')}>{c.medium}</button><button aria-pressed={risk === 'high'} onClick={() => setRisk('high')}>{c.high}</button></div></div>
      {error ? <InlineAlert message={error} /> : null}
      <TweakList items={items} busy={busy} onApply={(item) => void mutate('apply', item)} onRestore={(item) => void mutate('restore', item)} />
    </>
  )
}

function TweakList({ items, busy = null, onApply, onRestore }: { items: PlanItem[]; busy?: { id: string; action: 'apply' | 'restore' } | null; onApply?: (item: PlanItem) => void; onRestore?: (item: PlanItem) => void }) {
  return <section className="tweak-list">{items.map((item) => <TweakRow key={item.id} item={item} busy={busy ? (busy.id === item.id ? busy.action : 'other') : null} onApply={onApply} onRestore={onRestore} />)}</section>
}

function TweakRow({ item, busy, onApply, onRestore }: { item: PlanItem; busy: 'apply' | 'restore' | 'other' | null; onApply?: (item: PlanItem) => void; onRestore?: (item: PlanItem) => void }) {
  const { language } = useLanguage()
  const c = tweaksCopy[language]
  const detail = tweakDetail(item, language)
  return (
    <details className="tweak-row panel">
      <summary><div><span className="cell-sub">{localizePlan(item.category, language)}</span><h3>{localizePlan(item.name, language)}</h3></div><div className="tweak-change"><span>{localizePlan(item.current, language)}</span><b>→</b><span>{localizePlan(item.desired, language)}</span></div><span className={`risk-badge risk-badge--${item.risk_level}`}>{c.risk}: {c[item.risk_level]}</span><StatusDot tone={item.changed ? 'warning' : 'success'} label={item.changed ? c.willChange : item.restore_available ? c.rollbackReady : c.matches} /><ChevronDown className="tweak-disclosure" size={16} aria-hidden="true" /></summary>
      <div className="tweak-details"><div><span>{c.whatItDoes}</span><p>{detail.effect}</p></div><div><span>{c.benefit}</span><p>{detail.benefit}</p></div><div><span>{c.tradeoff}</span><p>{detail.risk}</p></div></div>
      <footer><div className="tweak-recovery"><span><Check size={13} />{item.manual_available ? c.autoBackup : c.noPersistentChange}</span><code>{item.id}</code></div>{onApply || onRestore ? <div className="tweak-actions">{item.restore_available && onRestore ? <Button variant="quiet" disabled={busy !== null} onClick={() => onRestore(item)}>{busy === 'restore' ? c.restoring : c.restoreTweak}</Button> : null}{item.changed && item.manual_available && onApply ? <Button variant="secondary" disabled={busy !== null} onClick={() => onApply(item)}>{busy === 'apply' ? c.applyingTweak : c.applyTweak}</Button> : null}{!item.manual_available ? <span>{c.sessionOnlyAction}</span> : !item.changed && !item.restore_available ? <span className="text-success">{c.noAction}</span> : null}</div> : null}</footer>
    </details>
  )
}

export function sortTweakItems(items: PlanItem[], mode: SortMode, language: 'ru' | 'en'): PlanItem[] {
  const result = [...items]
  if (mode === 'default') return result
  const byName = (left: PlanItem, right: PlanItem) => localizePlan(left.name, language).localeCompare(localizePlan(right.name, language), language)
  if (mode === 'name') return result.sort(byName)
  if (mode === 'risk') {
    const rank = { low: 0, medium: 1, high: 2 }
    return result.sort((left, right) => rank[right.risk_level] - rank[left.risk_level] || byName(left, right))
  }
  return result.sort((left, right) => Number(right.changed) - Number(left.changed) || Number(right.restore_available) - Number(left.restore_available) || byName(left, right))
}

function PresetCard({ id, selected, title, subtitle, description, count, risk, icon, onSelect }: { id: ProfileID; selected: boolean; title: string; subtitle: string; description: string; count: number; risk: string; icon: React.ReactNode; onSelect: (id: ProfileID) => void }) {
  const { language } = useLanguage()
  const c = tweaksCopy[language]
  return <button className={`preset-card ${selected ? 'preset-card--selected' : ''}`} aria-pressed={selected} onClick={() => onSelect(id)}><span className="preset-card__icon">{icon}</span><span className="preset-card__copy"><small>{subtitle}</small><strong>{title}</strong><p>{description}</p></span><span className="preset-card__meta"><b>{count}</b> {c.tweaksCount} · {risk}</span></button>
}

function localizePlan(value: string, language: 'ru' | 'en'): string {
  if (language === 'ru') return value
  const map: Record<string, string> = {
    'Игры': 'Gaming', 'Ввод': 'Input', 'Интерфейс': 'Interface', 'Питание': 'Power', 'Процессор': 'Processor', 'Накопители': 'Storage',
    'Разрешить автоматический Game Mode': 'Allow automatic Game Mode', 'Включить Game Mode': 'Enable Game Mode', 'Отключить фоновую запись Game DVR': 'Disable background Game DVR recording', 'Отключить фоновый захват экрана': 'Disable background screen capture', 'Отключить ускорение указателя': 'Disable pointer acceleration', 'Убрать первый порог ускорения мыши': 'Remove mouse acceleration threshold 1', 'Убрать второй порог ускорения мыши': 'Remove mouse acceleration threshold 2', 'Отключить прозрачность Windows': 'Disable Windows transparency', 'Отключить анимацию панели задач': 'Disable taskbar animations', 'Отключить анимацию сворачивания окон': 'Disable window minimize animations', 'Убрать задержку запуска автозагрузки после входа': 'Remove startup delay after sign-in', 'Убрать задержку открытия меню': 'Remove menu opening delay', 'Создать отдельную обратимую схему максимальной производительности': 'Create a separate reversible maximum-performance power plan', 'новая схема Luxury Optimization Max Performance': 'new Luxury Optimization Max Performance plan', '0 (отключено)': '0 (disabled)',
    'Создать отдельную обратимую схему Medium': 'Create a separate reversible Medium power plan', 'новая схема Luxury Optimization Medium': 'new Luxury Optimization Medium plan',
    'Создать отдельную обратимую схему Max': 'Create a separate reversible Max power plan',
  }
  return map[value] ?? value
}

function tweakDetail(item: PlanItem, language: 'ru' | 'en') {
  if (language === 'en') return { effect: item.effect, benefit: item.benefit, risk: item.risk }
  const detail = russianTweakMetadata[item.id]
  if (detail) return detail
  if (item.id.startsWith('power-')) return { effect: 'Меняет один поддерживаемый AC-параметр только в отдельной схеме Luxury.', benefit: 'Может сократить задержку перехода CPU или устройства из энергосбережения.', risk: 'Повышенный нагрев и энергопотребление; неподдерживаемые параметры пропускаются.' }
  if (item.id.startsWith('ethernet-')) return { effect: 'Отключает один параметр энергосбережения или модерации прерываний физического Ethernet-адаптера.', benefit: 'Может уменьшить пакетную обработку и задержку пробуждения проводной сети.', risk: 'Возможен рост нагрузки CPU, энергопотребления или падение пропускной способности.' }
  return { effect: item.effect, benefit: item.benefit, risk: item.risk }
}

const russianTweakMetadata: Record<string, { effect: string; benefit: string; risk: string }> = {
  'game-mode-allow': { effect: 'Разрешает Windows автоматически включать Game Mode для обнаруженных игр.', benefit: 'Может уменьшить влияние фоновых задач планировщика во время игры.', risk: 'Низкий: на части систем измеримого эффекта не будет.' },
  'game-mode-enable': { effect: 'Включает пользовательскую настройку Game Mode Windows.', benefit: 'Может отдать приоритет активной игре перед фоновыми приложениями.', risk: 'Низкий: фоновые приложения могут отвечать медленнее; прирост зависит от нагрузки.' },
  'capture-game-dvr': { effect: 'Отключает подготовку фоновой записи Game DVR.', benefit: 'Убирает ненужную нагрузку захвата на CPU, GPU и накопитель.', risk: 'Средний: фоновая запись и мгновенный повтор станут недоступны.' },
  'capture-app': { effect: 'Отключает фоновый захват экрана для текущего пользователя.', benefit: 'Убирает накладные расходы захвата и случайную запись.', risk: 'Средний: для записи функцию придётся включить обратно.' },
  'mouse-speed': { effect: 'Отключает ускорение указателя для текущего пользователя.', benefit: 'Делает физическое движение мыши более предсказуемым для прицеливания.', risk: 'Низкий: привычное ощущение курсора изменится; FPS не увеличивает.' },
  'mouse-threshold-1': { effect: 'Убирает первый legacy-порог ускорения мыши.', benefit: 'Сохраняет линейный ввод при отключённом ускорении.', risk: 'Низкий: меняется только ощущение мыши.' },
  'mouse-threshold-2': { effect: 'Убирает второй legacy-порог ускорения мыши.', benefit: 'Сохраняет линейный ввод при отключённом ускорении.', risk: 'Низкий: меняется только ощущение мыши.' },
  'ui-transparency': { effect: 'Отключает прозрачность Windows.', benefit: 'Немного снижает работу desktop compositor вне игры.', risk: 'Низкий: пропадёт визуальная прозрачность; игровой эффект обычно мал.' },
  'ui-taskbar-animation': { effect: 'Отключает анимации панели задач.', benefit: 'Делает действия оболочки мгновенными.', risk: 'Низкий: только визуальный компромисс, без гарантии прироста FPS.' },
  'ui-window-animation': { effect: 'Отключает анимации сворачивания и восстановления окон.', benefit: 'Уменьшает задержку оболочки при переключении из игры.', risk: 'Низкий: переходы окон станут резкими.' },
  'startup-delay': { effect: 'Убирает задержку запуска программ после входа.', benefit: 'Пользовательские программы стартуют раньше.', risk: 'Средний: сильнее одновременная нагрузка CPU и диска при входе.' },
  'menu-show-delay': { effect: 'Убирает задержку открытия классических меню.', benefit: 'Меню интерфейса отвечают сразу.', risk: 'Низкий: интерфейс может казаться резким; FPS не меняется.' },
  'power-plan': { effect: 'Создаёт отдельную обратимую схему максимальной производительности, не меняя исходную.', benefit: 'Может уменьшить задержки переходов частот CPU и энергосостояний устройств.', risk: 'Высокий: больше нагрев, шум и энергопотребление; использовать от сети.' },
}

const tweaksCopy = {
  en: {
    loading: 'Reading all supported tweaks…', title: 'Tweaks', description: 'Every available tweak, its exact change, expected benefit, risk and rollback in one place.', tweaks: 'Tweaks', profiles: 'Profiles', dismiss: 'Dismiss notification', allTweaks: 'All available tweaks', catalogDescription: 'Apply any tweak manually. A dedicated sealed backup is created automatically before every change.', search: 'Search tweaks', sort: 'Sort', sortDefault: 'Catalog order', sortName: 'Name', sortRisk: 'Risk: high first', sortStatus: 'Needs action first', riskFilter: 'Risk filter', allRisks: 'All risks', low: 'Low', medium: 'Medium', high: 'High', risk: 'Risk', willChange: 'Will change', matches: 'Already matches', rollbackReady: 'Rollback ready', whatItDoes: 'What it does', benefit: 'Expected benefit', tradeoff: 'Risk / trade-off', reversible: 'Exact rollback available', autoBackup: 'Automatic sealed backup before apply', noPersistentChange: 'No persistent system change', sessionOnlyAction: 'Available through a game session', notReversible: 'No rollback', applyTweak: 'Apply manually', applyingTweak: 'Applying…', restoreTweak: 'Roll back', restoring: 'Restoring…', noAction: 'No change needed', noChange: 'The tweak already matches the target state.', tweakApplied: (name: string) => `${name}: applied, verified and backed up.`, tweakRestored: (name: string) => `${name}: original state restored and verified.`, lite: 'LITE', liteSubtitle: 'Safe daily baseline', liteDescription: 'Six low-risk gaming and interface improvements with no power-plan changes.', liteRisk: 'low risk', mediumProfile: 'MEDIUM', mediumSubtitle: 'Balanced performance', mediumDescription: 'Lite plus capture/input tuning and 11 reviewed CPU power policies.', mediumRisk: 'moderate heat and power risk', max: 'MAX', maxSubtitle: 'Maximum supported optimization', maxDescription: 'All supported native CPU, storage, PCIe, USB, gaming and Ethernet actions.', maxRisk: 'higher heat and power risk', tweaksCount: 'tweaks', selectedPreset: 'Selected preset', liteIntent: 'Compatibility and stability first: only low-risk actions, with no power-plan or network mutation.', mediumIntent: 'A measured middle tier: capture/input tuning plus 11 reviewed CPU policies in a cloned AC plan.', maxIntent: 'All reviewed native CPU, storage, PCIe, USB and physical Ethernet actions. Cooling and AC power are required.', toChange: 'to change', checkpointRequired: 'Local Luxury restore point required', checkpointRequiredDetail: 'Create it with one click before Apply. It captures Registry Editor values and every optimizer-controlled state.', createCheckpoint: 'Create restore point', creating: 'Creating…', checkpointReady: 'Local restore point is ready', checkpointReadyDetail: (expires?: string) => expires ? `Valid until ${new Date(expires).toLocaleString('en-US')}. Apply also creates a fresh transactional backup.` : 'Apply also creates a fresh transactional backup.', ready: 'Ready', readyToApply: 'Preset is ready to apply.', createFirst: 'Create the local restore point first.', applyDetail: 'Apply creates another transactional backup, reads every target back and rolls back on mismatch.', applyPreset: (count: number) => `Apply preset · ${count} changes`, applyTitle: (id: ProfileID) => `Apply ${id === 'lite' ? 'Lite' : id === 'medium' ? 'Medium' : 'Max'} preset?`, transaction: 'The bounded transaction uses only the minimum required elevation:', exactTargets: (count: number) => `${count} exact targets`, checkpointConfirmed: 'Local Luxury restore point confirmed', transactionalBackup: 'Fresh transactional backup before mutation', nativeReadback: 'Native read-back and automatic rollback on mismatch', checkpointCreated: 'Local Luxury restore point created.', applied: 'Preset applied and every target verified.',
  },
  ru: {
    loading: 'Чтение всех поддерживаемых твиков…', title: 'ТВИКИ', description: 'Все доступные твики, точные изменения, ожидаемая польза, риски и откат в одном месте.', tweaks: 'Твики', profiles: 'Профили', dismiss: 'Закрыть уведомление', allTweaks: 'Все доступные твики', catalogDescription: 'Любой твик можно применить вручную. Перед каждым изменением автоматически создаётся отдельный sealed backup.', search: 'Поиск твиков', sort: 'Сортировка', sortDefault: 'Порядок каталога', sortName: 'По названию', sortRisk: 'Сначала высокий риск', sortStatus: 'Сначала требующие действий', riskFilter: 'Фильтр риска', allRisks: 'Все риски', low: 'Низкий', medium: 'Средний', high: 'Высокий', risk: 'Риск', willChange: 'Будет изменён', matches: 'Уже совпадает', rollbackReady: 'Откат готов', whatItDoes: 'Что делает', benefit: 'Ожидаемая польза', tradeoff: 'Риск / компромисс', reversible: 'Доступен точный откат', autoBackup: 'Автоматический sealed backup перед применением', noPersistentChange: 'Постоянные настройки системы не меняются', sessionOnlyAction: 'Доступно через игровую сессию', notReversible: 'Без отката', applyTweak: 'Применить вручную', applyingTweak: 'Применение…', restoreTweak: 'Откатить', restoring: 'Откат…', noAction: 'Изменения не нужны', noChange: 'Твик уже находится в нужном состоянии.', tweakApplied: (name: string) => `${name}: применён, проверен и сохранён для отката.`, tweakRestored: (name: string) => `${name}: исходное состояние возвращено и проверено.`, lite: 'LITE', liteSubtitle: 'Безопасная ежедневная база', liteDescription: 'Шесть низкорисковых игровых и интерфейсных улучшений без изменения схемы питания.', liteRisk: 'низкий риск', mediumProfile: 'MEDIUM', mediumSubtitle: 'Сбалансированная производительность', mediumDescription: 'Lite плюс настройка захвата/ввода и 11 проверенных CPU-политик питания.', mediumRisk: 'умеренный нагрев и расход энергии', max: 'MAX', maxSubtitle: 'Максимальная поддерживаемая оптимизация', maxDescription: 'Все поддерживаемые CPU, storage, PCIe, USB, игровые и Ethernet-действия.', maxRisk: 'повышенный нагрев и расход энергии', tweaksCount: 'твиков', selectedPreset: 'Выбранный пресет', liteIntent: 'Совместимость и стабильность прежде всего: только низкорисковые действия без изменения питания и сети.', mediumIntent: 'Средний уровень: захват/ввод и 11 проверенных CPU-политик в отдельной AC-схеме.', maxIntent: 'Все проверенные CPU, storage, PCIe, USB и Ethernet-действия. Нужны нормальное охлаждение и питание от сети.', toChange: 'будут изменены', checkpointRequired: 'Нужна локальная точка восстановления Luxury', checkpointRequiredDetail: 'Создайте её одной кнопкой до Apply. Она сохранит значения Редактора реестра и все состояния, которыми управляет оптимизатор.', createCheckpoint: 'Создать точку восстановления', creating: 'Создание…', checkpointReady: 'Локальная точка восстановления готова', checkpointReadyDetail: (expires?: string) => expires ? `Действует до ${new Date(expires).toLocaleString('ru-RU')}. Apply также создаст свежий transactional backup.` : 'Apply также создаст свежий transactional backup.', ready: 'Готово', readyToApply: 'Пресет готов к применению.', createFirst: 'Сначала создайте локальную точку восстановления.', applyDetail: 'Apply создаёт ещё один transactional backup, читает каждую цель обратно и откатывает несовпадение.', applyPreset: (count: number) => `Применить пресет · изменений ${count}`, applyTitle: (id: ProfileID) => `Применить пресет «${id === 'lite' ? 'Lite' : id === 'medium' ? 'Medium' : 'Max'}»?`, transaction: 'Ограниченная транзакция использует только минимально необходимые права:', exactTargets: (count: number) => `Точных целей: ${count}`, checkpointConfirmed: 'Локальная точка Luxury подтверждена', transactionalBackup: 'Свежий transactional backup до изменений', nativeReadback: 'Нативный read-back и автоматический откат при несовпадении', checkpointCreated: 'Локальная точка восстановления Luxury создана.', applied: 'Пресет применён, каждая цель проверена.',
  },
}
