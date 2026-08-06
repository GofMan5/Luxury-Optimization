import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './app/app'
import { BackendProvider } from './app/backend-context'
import { LanguageProvider } from './app/language-context'
import { UpdateProvider } from './app/update-context'
import './styles/global.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <LanguageProvider><BackendProvider><UpdateProvider><App /></UpdateProvider></BackendProvider></LanguageProvider>
  </StrictMode>,
)
