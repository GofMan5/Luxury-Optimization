import { lazy, StrictMode, Suspense } from 'react'
import { createRoot } from 'react-dom/client'
import App from './app/app'
import { BackendProvider } from './app/backend-context'
import { LanguageProvider } from './app/language-context'
import { UpdateProvider } from './app/update-context'
import './styles/global.css'

const StorageAnalyzerWindow = lazy(() => import('./features/storage/storage-analyzer').then((module) => ({ default: module.StorageAnalyzerWindow })))
const analyzerPath = new URLSearchParams(window.location.search).get('storage-analyzer')

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <LanguageProvider><BackendProvider>{analyzerPath && analyzerPath.length <= 4096
      ? <Suspense fallback={null}><StorageAnalyzerWindow initialPath={analyzerPath} /></Suspense>
      : <UpdateProvider><App /></UpdateProvider>}
    </BackendProvider></LanguageProvider>
  </StrictMode>,
)
