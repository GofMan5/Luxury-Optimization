import { useState } from 'react'
import { Activity, Cable, PlaySquare, ServerCog } from 'lucide-react'
import { useLanguage } from '../../app/language-context'
import { PageHeader } from '../../shared/ui/page-header'
import StartupScreen from '../startup/startup-screen'
import ServicesScreen from '../services/services-screen'
import NetworkScreen from '../network/network-screen'
import BackgroundScreen from '../background/background-screen'

type SystemSection = 'background' | 'startup' | 'services' | 'network'

export default function SystemScreen() {
  const { language } = useLanguage()
  const [section, setSection] = useState<SystemSection>('background')
  const copy = language === 'ru' ? {
    title: 'Система', description: 'Фоновая нагрузка, автозагрузка, системные службы и диагностика сети в одном месте. Опасные массовые изменения исключены.',
    background: 'Нагрузка', startup: 'Автозагрузка', services: 'Службы', network: 'Сеть',
  } : {
    title: 'System', description: 'Background load, startup, system services and network diagnostics in one place. Dangerous bulk mutations stay excluded.',
    background: 'Load', startup: 'Startup', services: 'Services', network: 'Network',
  }
  return (
    <div className="page">
      <PageHeader title={copy.title} description={copy.description} />
      <div className="system-tabs" role="tablist" aria-label={copy.title}>
        <button role="tab" aria-selected={section === 'background'} onClick={() => setSection('background')}><Activity size={17} />{copy.background}</button>
        <button role="tab" aria-selected={section === 'startup'} onClick={() => setSection('startup')}><PlaySquare size={17} />{copy.startup}</button>
        <button role="tab" aria-selected={section === 'services'} onClick={() => setSection('services')}><ServerCog size={17} />{copy.services}</button>
        <button role="tab" aria-selected={section === 'network'} onClick={() => setSection('network')}><Cable size={17} />{copy.network}</button>
      </div>
      <div role="tabpanel">
        {section === 'background' ? <BackgroundScreen onSection={setSection} /> : null}
        {section === 'startup' ? <StartupScreen embedded /> : null}
        {section === 'services' ? <ServicesScreen embedded /> : null}
        {section === 'network' ? <NetworkScreen embedded /> : null}
      </div>
    </div>
  )
}
