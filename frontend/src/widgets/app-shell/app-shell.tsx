import { CheckCircle2, Download, RefreshCw } from 'lucide-react'
import { useState, type PropsWithChildren } from 'react'
import type { RouteID } from '../../app/routes'
import { useBackend } from '../../app/backend-context'
import { useUpdates } from '../../app/update-context'
import { Sidebar } from './sidebar'
import { useLanguage } from '../../app/language-context'
import { Button } from '../../shared/ui/button'
import { ConfirmDialog } from '../../shared/ui/confirm-dialog'

export function AppShell({ route, onNavigate, children }: PropsWithChildren<{ route: RouteID; onNavigate: (route: RouteID) => void }>) {
  const { preview } = useBackend()
  const { language } = useLanguage()
  const { status, busy, progress, error, checkNow, install } = useUpdates()
  const [confirming, setConfirming] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const copy = language === 'ru' ? {
    preview: 'Предпросмотр в браузере · системные изменения имитируются', current: 'Актуальная версия', checking: 'Проверка…', check: 'Проверить обновления', available: (version: string) => `Доступна ${version}`, install: 'Установить', installing: (value: number | null) => value === null ? 'Установка…' : `Установка · ${value}%`, title: (version: string) => `Установить ${version}?`, description: 'Пакет будет скачан, проверен подписью и установлен с автоматическим перезапуском.', confirm: 'Скачать и установить', installed: 'Обновление установлено. Приложение перезапускается.', failed: 'Ошибка автообновления', dismiss: 'Скрыть',
  } : {
    preview: 'Browser preview · system changes are simulated', current: 'Up to date', checking: 'Checking…', check: 'Check for updates', available: (version: string) => `${version} available`, install: 'Install', installing: (value: number | null) => value === null ? 'Installing…' : `Installing · ${value}%`, title: (version: string) => `Install ${version}?`, description: 'The signed bundle will be downloaded, verified and installed, then the app restarts.', confirm: 'Download and install', installed: 'Update installed. The app is restarting.', failed: 'Automatic update failed', dismiss: 'Dismiss',
  }

  const check = async () => {
    setMessage(null)
    try { await checkNow() } catch { /* bounded context error is rendered below */ }
  }
  const apply = async () => {
    setMessage(null)
    try { setMessage((await install()) || copy.installed); setConfirming(false) } catch { /* bounded context error is rendered below */ }
  }

  return (
    <div className="app-frame">
      <Sidebar route={route} onNavigate={onNavigate} />
      <main className="workspace">
        <header className="workspace-bar">
          <span className="workspace-bar__route">{routeLabel(route, language)}</span>
          <div className="workspace-bar__update">
            {error ? <span className="update-error" title={error}>{copy.failed}</span> : null}
            {message ? <span className="update-success" role="status"><CheckCircle2 size={14} />{message}<button onClick={() => setMessage(null)} aria-label={copy.dismiss}>×</button></span> : null}
            <button className={`update-status ${status?.update_ready ? 'update-status--ready' : ''}`} disabled={busy !== null} title={status?.update_ready ? copy.available(status.latest ?? '') : copy.check} onClick={() => void check()}>
              <RefreshCw className={busy === 'check' ? 'spinner' : ''} size={14} />
              <span>{busy === 'check' ? copy.checking : status?.update_ready ? copy.available(status.latest ?? '') : status ? `${copy.current} · ${status.current}` : copy.check}</span>
            </button>
            {status?.update_ready ? <Button variant="quiet" disabled={busy !== null} onClick={() => setConfirming(true)}><Download size={14} />{busy === 'install' ? copy.installing(progress) : copy.install}</Button> : null}
          </div>
        </header>
        {preview ? <div className="preview-banner">{copy.preview}</div> : null}
        {children}
        <ConfirmDialog open={confirming} title={copy.title(status?.latest ?? '')} description={<p>{copy.description}</p>} confirmLabel={copy.confirm} busy={busy === 'install'} onCancel={() => setConfirming(false)} onConfirm={() => void apply()} />
      </main>
    </div>
  )
}

function routeLabel(route: RouteID, language: 'ru' | 'en'): string {
  const labels: Record<RouteID, [string, string]> = {
    overview: ['Главная', 'Home'], profiles: ['Оптимизация', 'Optimization'], games: ['Игры', 'Games'], benchmarks: ['Измерения', 'Measurements'], system: ['Инструменты', 'Tools'], restore: ['Восстановление', 'Recovery'],
  }
  return labels[route][language === 'ru' ? 0 : 1]
}
