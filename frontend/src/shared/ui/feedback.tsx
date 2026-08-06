import { AlertTriangle, LoaderCircle, RotateCw } from 'lucide-react'
import { Button } from './button'
import { useLanguage } from '../../app/language-context'

export function LoadingState({ label = 'Loading current system state…' }: { label?: string }) {
  const { language } = useLanguage()
  const resolved = label === 'Loading current system state…' && language === 'ru' ? 'Загрузка текущего состояния системы…' : label
  return <div className="state-message" role="status" aria-live="polite"><LoaderCircle className="spinner" size={20} aria-hidden="true" /><span>{resolved}</span></div>
}

export function InlineAlert({ message, onRetry }: { message: string; onRetry?: () => void }) {
  const { language } = useLanguage()
  return (
    <div className="inline-alert" role="alert">
      <AlertTriangle size={19} aria-hidden="true" />
      <span>{message}</span>
      {onRetry ? <Button variant="quiet" onClick={onRetry}><RotateCw size={15} />{language === 'ru' ? 'Повторить' : 'Retry'}</Button> : null}
    </div>
  )
}

export function EmptyState({ title, detail }: { title: string; detail: string }) {
  return <div className="empty-state"><strong>{title}</strong><span>{detail}</span></div>
}
