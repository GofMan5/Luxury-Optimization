import { ArchiveRestore, BarChart3, Gamepad2, Gauge, MonitorCog, Rocket, SlidersHorizontal } from 'lucide-react'
import type { ComponentType } from 'react'
import type { RouteID } from '../../app/routes'
import { useLanguage } from '../../app/language-context'

interface NavigationItem {
  id: RouteID
  label: { en: string; ru: string }
  icon: ComponentType<{ size?: number; strokeWidth?: number; 'aria-hidden'?: boolean }>
}

const navigation: NavigationItem[] = [
  { id: 'overview', label: { en: 'Overview', ru: 'Обзор' }, icon: Gauge },
  { id: 'profiles', label: { en: 'Tweaks', ru: 'ТВИКИ' }, icon: SlidersHorizontal },
  { id: 'games', label: { en: 'Games', ru: 'Игры' }, icon: Gamepad2 },
  { id: 'benchmarks', label: { en: 'Benchmarks', ru: 'Замеры' }, icon: BarChart3 },
  { id: 'system', label: { en: 'System', ru: 'Система' }, icon: MonitorCog },
  { id: 'restore', label: { en: 'Restore', ru: 'Восстановление' }, icon: ArchiveRestore },
  { id: 'updates', label: { en: 'Updates', ru: 'Обновления' }, icon: Rocket },
]

export function Sidebar({ route, onNavigate }: { route: RouteID; onNavigate: (route: RouteID) => void }) {
  const { language, setLanguage } = useLanguage()
  return (
    <aside className="sidebar">
      <div className="brand" aria-label="Luxury Optimization">
        <span className="brand__mark" aria-hidden="true">L</span>
        <span className="brand__text"><strong>Luxury</strong><small>Optimization</small></span>
      </div>
      <nav className="sidebar__nav" aria-label="Primary navigation">
        {navigation.map(({ id, label, icon: Icon }) => (
          <button key={id} className={`nav-item ${route === id ? 'nav-item--active' : ''}`} aria-current={route === id ? 'page' : undefined} title={label[language]} onClick={() => onNavigate(id)}>
            <Icon size={19} strokeWidth={1.8} aria-hidden={true} /><span>{label[language]}</span>
          </button>
        ))}
      </nav>
      <div className="language-switch" role="group" aria-label={language === 'ru' ? 'Язык интерфейса' : 'Interface language'}>
        <button aria-pressed={language === 'ru'} onClick={() => setLanguage('ru')}>RU</button>
        <button aria-pressed={language === 'en'} onClick={() => setLanguage('en')}>EN</button>
      </div>
    </aside>
  )
}
