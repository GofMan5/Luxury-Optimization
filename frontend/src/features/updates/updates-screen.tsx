import { useState } from 'react'
import { CheckCircle2, Download, RefreshCw } from 'lucide-react'
import { useLanguage } from '../../app/language-context'
import { useUpdates } from '../../app/update-context'
import { Button } from '../../shared/ui/button'
import { ConfirmDialog } from '../../shared/ui/confirm-dialog'
import { InlineAlert, LoadingState } from '../../shared/ui/feedback'
import { PageHeader } from '../../shared/ui/page-header'
import { StatusDot } from '../../shared/ui/status'

export default function UpdatesScreen() {
  const { language } = useLanguage()
  const { status, loading, busy, progress, error, checkNow, install } = useUpdates()
  const c = updatesCopy[language]
  const [confirming, setConfirming] = useState(false)
  const [message, setMessage] = useState<string | null>(null)

  const check = async () => {
    setMessage(null)
    try {
      const next = await checkNow()
      setMessage(next.update_ready ? c.ready(next.latest ?? '') : c.current(next.current))
    } catch { /* context renders the bounded error */ }
  }

  const apply = async () => {
    setMessage(null)
    try {
      const result = await install()
      setMessage(result || c.installed)
      setConfirming(false)
    } catch { /* context renders the bounded error */ }
  }

  if (loading) return <div className="page"><LoadingState label={c.loading} /></div>
  return (
    <div className="page">
      <PageHeader title={c.title} description={c.description} actions={<Button variant="secondary" disabled={busy !== null} onClick={() => void check()}><RefreshCw size={16} />{busy === 'check' ? c.checking : c.check}</Button>} />
      {error ? <InlineAlert message={error} onRetry={() => void check()} /> : null}
      {message ? <div className="success-notice" role="status"><CheckCircle2 size={18} />{message}<button onClick={() => setMessage(null)} aria-label={c.dismiss}>×</button></div> : null}
      {status ? <section className="update-hero panel panel--gold"><div className="update-hero__version"><span>{c.installedVersion}</span><strong>{status.current}</strong><StatusDot tone={status.update_ready ? 'warning' : 'success'} label={status.update_ready ? c.available(status.latest ?? '') : c.upToDate} /><small>{c.lastCheck}: {status.last_check ? new Date(status.last_check).toLocaleString(language === 'ru' ? 'ru-RU' : 'en-US') : c.never}</small></div><div className="update-hero__actions"><Button variant="primary" disabled={!status.update_ready || busy !== null} onClick={() => setConfirming(true)}><Download size={16} />{busy === 'install' && progress !== null ? c.downloading(progress) : c.install}</Button></div></section> : null}
      <ConfirmDialog open={confirming} title={c.installTitle(status?.latest ?? '')} description={<><p>{c.installDescription}</p><ul><li>{c.rule1}</li><li>{c.rule2}</li><li>{c.rule3}</li></ul></>} confirmLabel={c.confirm} busy={busy === 'install'} onCancel={() => setConfirming(false)} onConfirm={() => void apply()} />
    </div>
  )
}

const updatesCopy = {
  en: {
    loading: 'Checking for signed updates…', title: 'Updates', description: 'Automatic checks use signed GitHub Releases in the supported 1.0.x line.', check: 'Check now', checking: 'Checking…', ready: (version: string) => `${version} is ready to install.`, current: (version: string) => `Version ${version} is current.`, installed: 'Update installed. Luxury Optimization is restarting.', dismiss: 'Dismiss notification', installedVersion: 'Installed version', available: (version: string) => `${version} available`, upToDate: 'Up to date', lastCheck: 'Last check', never: 'Never', install: 'Install update', downloading: (value: number) => `Installing · ${value}%`, installTitle: (version: string) => `Install ${version || 'available update'}?`, installDescription: 'Tauri verifies the signed platform bundle before installation.', rule1: 'Only the 1.0.x release line is accepted', rule2: 'A failed signature check leaves the current build untouched', rule3: 'The app restarts automatically after installation', confirm: 'Download and install',
  },
  ru: {
    loading: 'Проверка подписанных обновлений…', title: 'Обновления', description: 'Автопроверка использует подписанные GitHub Releases поддерживаемой ветки 1.0.x.', check: 'Проверить', checking: 'Проверка…', ready: (version: string) => `Версия ${version} готова к установке.`, current: (version: string) => `Установлена актуальная версия ${version}.`, installed: 'Обновление установлено. Luxury Optimization перезапускается.', dismiss: 'Закрыть уведомление', installedVersion: 'Установленная версия', available: (version: string) => `Доступна ${version}`, upToDate: 'Актуальная версия', lastCheck: 'Последняя проверка', never: 'Ещё не было', install: 'Установить', downloading: (value: number) => `Установка · ${value}%`, installTitle: (version: string) => `Установить ${version || 'доступное обновление'}?`, installDescription: 'Tauri проверит подпись сборки для этой платформы до установки.', rule1: 'Принимается только ветка релизов 1.0.x', rule2: 'Ошибка подписи не затронет текущую сборку', rule3: 'После установки приложение перезапустится автоматически', confirm: 'Скачать и установить',
  },
}
