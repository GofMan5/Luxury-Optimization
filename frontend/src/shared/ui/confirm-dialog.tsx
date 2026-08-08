import { useEffect, useRef, type ReactNode } from 'react'
import { Button } from './button'
import { useLanguage } from '../../app/language-context'
import { dialogFocusTarget } from './dialog-focus'

interface ConfirmDialogProps {
  open: boolean
  title: string
  description: ReactNode
  confirmLabel: string
  busy?: boolean
  danger?: boolean
	confirmDisabled?: boolean
  onCancel: () => void
  onConfirm: () => void
}

export function ConfirmDialog({ open, title, description, confirmLabel, busy = false, danger = false, confirmDisabled = false, onCancel, onConfirm }: ConfirmDialogProps) {
  const { language } = useLanguage()
  const cancelRef = useRef<HTMLButtonElement>(null)
  const dialogRef = useRef<HTMLElement>(null)
  const cancelHandler = useRef(onCancel)
  const busyState = useRef(busy)
  cancelHandler.current = onCancel
  busyState.current = busy
  useEffect(() => {
    if (!open) return
    const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null
    cancelRef.current?.focus()
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !busyState.current) { event.preventDefault(); cancelHandler.current(); return }
      if (event.key !== 'Tab') return
      const controls = Array.from(dialogRef.current?.querySelectorAll<HTMLElement>('button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])') ?? [])
      if (controls.length === 0) return
      const target = dialogFocusTarget(controls.indexOf(document.activeElement as HTMLElement), controls.length - 1, event.shiftKey)
      if (target !== null) { event.preventDefault(); controls[target]?.focus() }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => { document.removeEventListener('keydown', onKeyDown); previous?.focus() }
  }, [open])
  if (!open) return null
  return (
    <div className="dialog-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget && !busy) onCancel() }}>
      <section ref={dialogRef} className="dialog" role="alertdialog" aria-modal="true" aria-labelledby="dialog-title" aria-describedby="dialog-description">
        <h2 id="dialog-title">{title}</h2>
        <div id="dialog-description" className="dialog__description">{description}</div>
        <div className="dialog__actions">
          <Button ref={cancelRef} variant="quiet" disabled={busy} onClick={onCancel}>{language === 'ru' ? 'Отмена' : 'Cancel'}</Button>
          <Button variant={danger ? 'danger' : 'primary'} disabled={busy || confirmDisabled} onClick={onConfirm}>{busy ? (language === 'ru' ? 'Выполняется…' : 'Working…') : confirmLabel}</Button>
        </div>
      </section>
    </div>
  )
}
