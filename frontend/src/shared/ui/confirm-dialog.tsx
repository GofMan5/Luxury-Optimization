import { useEffect, useRef, type ReactNode } from 'react'
import { Button } from './button'
import { useLanguage } from '../../app/language-context'

interface ConfirmDialogProps {
  open: boolean
  title: string
  description: ReactNode
  confirmLabel: string
  busy?: boolean
  danger?: boolean
  onCancel: () => void
  onConfirm: () => void
}

export function ConfirmDialog({ open, title, description, confirmLabel, busy = false, danger = false, onCancel, onConfirm }: ConfirmDialogProps) {
  const { language } = useLanguage()
  const cancelRef = useRef<HTMLButtonElement>(null)
  useEffect(() => {
    if (open) cancelRef.current?.focus()
  }, [open])
  if (!open) return null
  return (
    <div className="dialog-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget && !busy) onCancel() }}>
      <section className="dialog" role="alertdialog" aria-modal="true" aria-labelledby="dialog-title" aria-describedby="dialog-description">
        <h2 id="dialog-title">{title}</h2>
        <div id="dialog-description" className="dialog__description">{description}</div>
        <div className="dialog__actions">
          <Button ref={cancelRef} variant="quiet" disabled={busy} onClick={onCancel}>{language === 'ru' ? 'Отмена' : 'Cancel'}</Button>
          <Button variant={danger ? 'danger' : 'primary'} disabled={busy} onClick={onConfirm}>{busy ? (language === 'ru' ? 'Выполняется…' : 'Working…') : confirmLabel}</Button>
        </div>
      </section>
    </div>
  )
}
