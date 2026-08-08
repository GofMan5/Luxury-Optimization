import { BarChart3, Cable, HardDrive } from 'lucide-react'
import { useState } from 'react'
import { useLanguage } from '../../app/language-context'
import { PageHeader } from '../../shared/ui/page-header'
import BenchmarkScreen from '../benchmarks/benchmark-screen'
import NetworkScreen from '../network/network-screen'
import StorageScreen from '../storage/storage-screen'

type MeasurementSection = 'benchmark' | 'network' | 'storage'

export default function MeasurementsScreen() {
  const { language } = useLanguage()
  const [section, setSection] = useState<MeasurementSection>('benchmark')
  const copy = language === 'ru' ? {
    title: 'Измерения', description: 'Сравнивайте FPS и frametime, проверяйте сеть и накопители одинаковыми повторяемыми тестами.', benchmark: 'Игровой результат', network: 'Сеть', storage: 'Накопители',
  } : {
    title: 'Measurements', description: 'Compare FPS and frametime, then test network and storage with repeatable inputs.', benchmark: 'Game result', network: 'Network', storage: 'Storage',
  }
  return <div className="page">
    <PageHeader title={copy.title} description={copy.description} />
    <div className="system-tabs" role="tablist" aria-label={copy.title}>
      <button role="tab" aria-selected={section === 'benchmark'} onClick={() => setSection('benchmark')}><BarChart3 size={17} />{copy.benchmark}</button>
      <button role="tab" aria-selected={section === 'network'} onClick={() => setSection('network')}><Cable size={17} />{copy.network}</button>
      <button role="tab" aria-selected={section === 'storage'} onClick={() => setSection('storage')}><HardDrive size={17} />{copy.storage}</button>
    </div>
    <div role="tabpanel">
      {section === 'benchmark' ? <BenchmarkScreen embedded /> : null}
      {section === 'network' ? <NetworkScreen embedded /> : null}
      {section === 'storage' ? <StorageScreen /> : null}
    </div>
  </div>
}
