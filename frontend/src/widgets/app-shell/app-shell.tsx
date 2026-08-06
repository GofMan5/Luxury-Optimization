import type { PropsWithChildren } from 'react'
import type { RouteID } from '../../app/routes'
import { useBackend } from '../../app/backend-context'
import { Sidebar } from './sidebar'
import { useLanguage } from '../../app/language-context'

export function AppShell({ route, onNavigate, children }: PropsWithChildren<{ route: RouteID; onNavigate: (route: RouteID) => void }>) {
  const { preview } = useBackend()
  const { language } = useLanguage()

  return (
    <div className="app-frame">
      <Sidebar route={route} onNavigate={onNavigate} />
      <main className="workspace">
        {preview ? <div className="preview-banner">{language === 'ru' ? 'Предпросмотр в браузере · системные изменения имитируются' : 'Browser preview · system changes are simulated'}</div> : null}
        {children}
      </main>
    </div>
  )
}
