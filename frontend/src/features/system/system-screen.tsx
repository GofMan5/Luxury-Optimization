import { useState } from 'react'
import { Activity, PlaySquare, ServerCog } from 'lucide-react'
import { useLanguage } from '../../app/language-context'
import { PageHeader } from '../../shared/ui/page-header'
import StartupScreen from '../startup/startup-screen'
import ServicesScreen from '../services/services-screen'
import BackgroundScreen from '../background/background-screen'

type SystemSection = 'background' | 'startup' | 'services'

export default function SystemScreen() {
  const { language } = useLanguage()
  const [section, setSection] = useState<SystemSection>('background')
  const copy = language === 'ru' ? {
    title: 'Инструменты', description: 'Найдите подтверждённую фоновую нагрузку и управляйте только точными обратимыми целями.',
    background: 'Нагрузка', startup: 'Автозагрузка', services: 'Службы',
  } : {
    title: 'Tools', description: 'Find measured background load and manage only exact reversible targets.',
    background: 'Load', startup: 'Startup', services: 'Services',
  }
  return (
    <div className="page">
      <PageHeader title={copy.title} description={copy.description} />
      <div className="system-tabs" role="tablist" aria-label={copy.title}>
        <button role="tab" aria-selected={section === 'background'} onClick={() => setSection('background')}><Activity size={17} />{copy.background}</button>
        <button role="tab" aria-selected={section === 'startup'} onClick={() => setSection('startup')}><PlaySquare size={17} />{copy.startup}</button>
        <button role="tab" aria-selected={section === 'services'} onClick={() => setSection('services')}><ServerCog size={17} />{copy.services}</button>
      </div>
      <div role="tabpanel">
        {section === 'background' ? <BackgroundScreen onSection={setSection} /> : null}
        {section === 'startup' ? <StartupScreen embedded /> : null}
        {section === 'services' ? <ServicesScreen embedded /> : null}
      </div>
    </div>
  )
}
