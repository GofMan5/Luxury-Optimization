import { useCallback, useMemo, useState } from 'react'
import { ArrowRight, BarChart3, RotateCw, RotateCcw, ShieldCheck, Sparkles } from 'lucide-react'
import type { RouteID } from '../../app/routes'
import { useBackend } from '../../app/backend-context'
import type { Audit, CleanResult, Plan } from '../../shared/contracts/domain'
import { useBackendResource } from '../../shared/hooks/use-backend-resource'
import { Button } from '../../shared/ui/button'
import { ConfirmDialog } from '../../shared/ui/confirm-dialog'
import { InlineAlert, LoadingState } from '../../shared/ui/feedback'
import { PageHeader } from '../../shared/ui/page-header'
import { StatusDot } from '../../shared/ui/status'
import { useLanguage } from '../../app/language-context'

interface OverviewData { audit: Audit; plan: Plan }

export default function OverviewScreen({ onNavigate }: { onNavigate: (route: RouteID) => void }) {
  const { client } = useBackend()
  const { language } = useLanguage()
  const c = overviewCopy[language]
  const load = useCallback(async (signal: AbortSignal): Promise<OverviewData> => {
    const [audit, plan] = await Promise.all([
      client.call<Audit>('optimization.audit', {}, signal),
      client.call<Plan>('optimization.plan', { profile: 'lite' }, signal),
    ])
    return { audit, plan }
  }, [client])
  const resource = useBackendResource(load, [load])
  const [cleanOpen, setCleanOpen] = useState(false)
  const [cleaning, setCleaning] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  const changed = resource.data?.plan.items.filter((item) => item.changed).length ?? 0
  const total = resource.data?.plan.items.length ?? 0
  const readiness = total === 0 ? 100 : Math.round(((total - changed) / total) * 100)
  const ringStyle = useMemo(() => ({ '--score': `${readiness}%` }) as React.CSSProperties, [readiness])
  const refresh = () => { client.invalidate('optimization.'); resource.refresh() }

  const clean = async () => {
    setCleaning(true)
    try {
      const result = await client.call<CleanResult>('cleanup.run', { days: 2 })
      setNotice(c.cleanupResult(result.files, formatBytes(result.bytes), result.skipped))
      setCleanOpen(false)
    } catch (error) {
      setNotice(error instanceof Error ? error.message : String(error))
    } finally { setCleaning(false) }
  }

  if (resource.loading && !resource.data) return <div className="page"><LoadingState label={c.loading} /></div>
  if (resource.error && !resource.data) return <div className="page"><PageHeader title={c.title} description={c.shortDescription} /><InlineAlert message={resource.error} onRetry={refresh} /></div>
  const data = resource.data!

  return (
    <div className="page">
      <PageHeader
        title={c.title}
        description={c.description}
        actions={<StatusDot tone="success" label={data.audit.hardware.os.caption} />}
      />
      {resource.error ? <InlineAlert message={resource.error} onRetry={refresh} /> : null}
      {notice ? <div className="success-notice" role="status"><ShieldCheck size={18} />{notice}<button onClick={() => setNotice(null)} aria-label="Dismiss notification">×</button></div> : null}

      <section className="command-center panel panel--gold">
        <div className="command-center__copy">
          <span className="eyebrow">{c.currentState}</span>
          <h2>{changed === 0 ? c.baselineReady : c.ready}</h2>
          <p>{changed === 0 ? c.allMatch : c.differ(changed)}</p>
          <div className="command-center__actions">
            <Button variant="primary" onClick={() => onNavigate('profiles')}>{c.review} <ArrowRight size={16} /></Button>
            <Button variant="secondary" onClick={refresh}><RotateCw size={16} />{c.audit}</Button>
          </div>
        </div>
        <div className="readiness-meter" style={ringStyle} aria-label={c.readinessLabel(readiness)}><strong>{readiness}%</strong><span>{c.targetsMatch}</span></div>
        <dl className="command-center__facts">
          <div><dt>{c.powerPlan}</dt><dd>{data.audit.active_power_guid ? c.detected : c.unavailable}</dd></div>
          <div><dt>{c.elevation}</dt><dd>{data.audit.administrator ? c.administrator : c.onDemand}</dd></div>
          <div><dt>{c.lastAudit}</dt><dd>{formatRelative(data.audit.generated_at, language)}</dd></div>
          <div><dt>{c.findings}</dt><dd className={data.audit.optimization_findings.length ? 'text-warning' : 'text-success'}>{data.audit.optimization_findings.length}</dd></div>
        </dl>
      </section>

      <ol className="workflow-strip" aria-label={c.workflowLabel}>
        <li className="workflow-strip__active"><span>1</span><div><strong>{c.stepAudit}</strong><small>{c.stepAuditDetail}</small></div></li>
        <li><span>2</span><div><strong>{c.stepApply}</strong><small>{c.stepApplyDetail}</small></div></li>
        <li><span>3</span><div><strong>{c.stepMeasure}</strong><small>{c.stepMeasureDetail}</small></div><Button variant="quiet" onClick={() => onNavigate('benchmarks')}><BarChart3 size={14} />{c.openMeasurements}</Button></li>
        <li><span>4</span><div><strong>{c.stepRestore}</strong><small>{c.stepRestoreDetail}</small></div><Button variant="quiet" onClick={() => onNavigate('restore')}><RotateCcw size={14} />{c.openRecovery}</Button></li>
      </ol>

      <div className="section-heading"><div><h2>{c.capabilities}</h2><p>{c.capabilitiesDescription}</p></div></div>
      <section className="panel capability-panel">
        <ul className="notice-list">
          {data.audit.capabilities.map((capability) => (
            <li key={capability.id}>
              <span className={`notice-list__dot ${capability.available ? '' : 'notice-list__dot--muted'}`} aria-hidden="true" />
              <strong>{capabilityLabel(capability.id, language)}</strong>
              <span>{capabilityDetail(capability.id, capability.detail, language)}</span>
              <StatusDot tone={capability.available ? 'success' : 'muted'} label={capability.available ? localizeMode(capability.mode, language) : c.skipped} />
            </li>
          ))}
        </ul>
      </section>

      <section className="overview-session panel">
        <div><Sparkles size={19} aria-hidden="true" /><div><h2>{c.sessionTools}</h2><p>{c.sessionDescription}</p></div></div>
        <div className="overview-session__actions"><Button variant="quiet" onClick={() => setCleanOpen(true)}>{c.clean}</Button></div>
      </section>

      <ConfirmDialog open={cleanOpen} title={c.cleanTitle} description={<><p>{c.cleanDescription}</p><ul><li>{c.cleanRule1}</li><li>{c.cleanRule2}</li><li>{c.cleanRule3}</li></ul></>} confirmLabel={c.runCleanup} busy={cleaning} onCancel={() => setCleanOpen(false)} onConfirm={() => void clean()} />
    </div>
  )
}

function formatRelative(value: string, language: 'ru' | 'en'): string {
  const seconds = Math.max(0, Math.floor((Date.now() - new Date(value).getTime()) / 1000))
  if (seconds < 60) return language === 'ru' ? 'Только что' : 'Just now'
  if (seconds < 3600) return language === 'ru' ? `${Math.floor(seconds / 60)} мин назад` : `${Math.floor(seconds / 60)}m ago`
  return new Date(value).toLocaleString(language === 'ru' ? 'ru-RU' : 'en-US')
}

function formatBytes(bytes: number): string {
  return bytes < 1024 * 1024 ? `${Math.round(bytes / 1024)} KB` : `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function capabilityLabel(value: string, language: 'ru' | 'en'): string {
  const labels: Record<string, [string, string]> = {
    'persistent-profile': ['Persistent Profile', 'Постоянный профиль'],
    'game-boost': ['Game Boost', 'Игровой буст'],
    'game-discovery': ['Game Discovery', 'Поиск игр'],
    startup: ['Startup', 'Автозагрузка'],
    services: ['Services', 'Службы'],
    'self-update': ['Self Update', 'Автообновление'],
  }
  const label = labels[value]
  return label ? label[language === 'ru' ? 1 : 0] : value
}

function localizeMode(value: string, language: 'ru' | 'en'): string {
  if (language === 'en') return value
  return ({ reversible: 'обратимо', session: 'сессионно', 'read-only': 'только чтение', 'opt-in': 'по выбору' } as Record<string, string>)[value] ?? value
}

function capabilityDetail(id: string, fallback: string, language: 'ru' | 'en'): string {
  const details: Record<string, [string, string]> = {
    'persistent-profile': ['Registry, mouse, power and supported Ethernet use backup, apply, read-back and rollback', 'Registry, мышь, питание и поддерживаемый Ethernet: backup, применение, read-back и откат'],
    'game-boost': ['The game stays non-elevated; system state is restored after exit', 'Игра остаётся без повышения прав; состояние системы возвращается после выхода'],
    'game-discovery': ['Read-only discovery of supported game libraries', 'Поиск поддерживаемых библиотек только для чтения'],
    startup: ['Current-user startup values are backed up and restored exactly', 'Записи автозагрузки пользователя точно сохраняются и восстанавливаются'],
    services: ['Service inventory is read-only', 'Список служб доступен только для чтения'],
    'self-update': ['The matching release is verified before replacement', 'Подходящий релиз проверяется до замены'],
  }
  const detail = details[id]
  return detail ? detail[language === 'ru' ? 1 : 0] : fallback
}

const overviewCopy = {
  en: {
    loading: 'Loading current system state…', title: 'Your PC', shortDescription: 'Measured changes, exact rollback, no placebo presets.', description: 'See what matters, apply only supported changes and verify the result.', currentState: 'Recommended next step', baselineReady: 'Lite baseline already matches', ready: 'Review the safe recommended changes', allMatch: 'No Lite changes are currently required. Measure a game before trying a stronger profile.', differ: (count: number) => `${count} supported Lite targets differ. Review the exact changes before applying.`, review: 'Review optimization', audit: 'Refresh audit', readinessLabel: (score: number) => `Lite readiness: ${score} percent`, targetsMatch: 'Lite readiness', powerPlan: 'Power plan', detected: 'Detected', unavailable: 'Unavailable', elevation: 'Access', administrator: 'Administrator', onDemand: 'Requested only when applying', lastAudit: 'Audit', findings: 'Findings', workflowLabel: 'Optimization workflow', stepAudit: 'Audit', stepAuditDetail: 'Hardware and current state', stepApply: 'Apply', stepApplyDetail: 'Preview, checkpoint, read-back', stepMeasure: 'Measure', stepMeasureDetail: 'Compare repeatable runs', stepRestore: 'Restore', stepRestoreDetail: 'Exact saved state', openMeasurements: 'Open', openRecovery: 'Open', capabilities: 'Supported on this PC', capabilitiesDescription: 'Unavailable controls are reported and skipped without partial mutation.', skipped: 'Skipped', sessionTools: 'Session tools', sessionDescription: 'Game boost is available without elevating the game. Cleanup is limited to old application temp files.', clean: 'Clean old temp files', cleanTitle: 'Clean old temporary files?', cleanDescription: "Only files in Luxury Optimization's bounded temp scope older than 2 days are eligible.", cleanRule1: 'No user documents or game files', cleanRule2: 'Reparse points are skipped', cleanRule3: 'Elevated cleanup is refused', runCleanup: 'Run cleanup', cleanupResult: (files: number, size: string, skipped: number) => `Cleanup removed ${files} files and reclaimed ${size}; ${skipped} items were skipped safely.`,
  },
  ru: {
    loading: 'Загрузка текущего состояния системы…', title: 'Ваш ПК', shortDescription: 'Измеримые изменения, точный откат, никаких плацебо-твиков.', description: 'Сразу видно, что делать дальше, что изменится и как проверить результат.', currentState: 'Рекомендуемый следующий шаг', baselineReady: 'Базовый профиль Lite уже настроен', ready: 'Проверьте безопасные рекомендации', allMatch: 'Для Lite сейчас ничего менять не нужно. Сначала замерьте игру перед более сильным профилем.', differ: (count: number) => `Для Lite доступно изменений: ${count}. Проверьте точный список перед применением.`, review: 'Открыть оптимизацию', audit: 'Обновить аудит', readinessLabel: (score: number) => `Готовность Lite: ${score} процентов`, targetsMatch: 'Готовность Lite', powerPlan: 'Схема питания', detected: 'Определена', unavailable: 'Недоступна', elevation: 'Доступ', administrator: 'Администратор', onDemand: 'Запрос только при применении', lastAudit: 'Аудит', findings: 'Рекомендации', workflowLabel: 'Путь оптимизации', stepAudit: 'Аудит', stepAuditDetail: 'Железо и текущее состояние', stepApply: 'Применение', stepApplyDetail: 'Preview, checkpoint, read-back', stepMeasure: 'Замер', stepMeasureDetail: 'Сравнение повторяемых прогонов', stepRestore: 'Откат', stepRestoreDetail: 'Точное сохранённое состояние', openMeasurements: 'Открыть', openRecovery: 'Открыть', capabilities: 'Поддерживается на этом ПК', capabilitiesDescription: 'Недоступные настройки явно пропускаются без частичных изменений.', skipped: 'Пропущено', sessionTools: 'Сессионные инструменты', sessionDescription: 'Игровой буст не повышает права игры. Очистка ограничена старыми временными файлами приложения.', clean: 'Очистить старый кеш', cleanTitle: 'Очистить старые временные файлы?', cleanDescription: 'Под удаление попадают только файлы в ограниченной temp-области Luxury Optimization старше 2 дней.', cleanRule1: 'Документы пользователя и файлы игр не затрагиваются', cleanRule2: 'Точки повторной обработки пропускаются', cleanRule3: 'Очистка с правами администратора запрещена', runCleanup: 'Запустить очистку', cleanupResult: (files: number, size: string, skipped: number) => `Удалено файлов: ${files}, освобождено ${size}; безопасно пропущено: ${skipped}.`,
  },
}
