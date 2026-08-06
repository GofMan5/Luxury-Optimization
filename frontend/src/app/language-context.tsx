import { createContext, useContext, useEffect, useMemo, useState, type PropsWithChildren } from 'react'

export type Language = 'ru' | 'en'

interface LanguageContextValue {
  language: Language
  setLanguage: (language: Language) => void
}

const storageKey = 'luxury-optimization.ui.v1'
const LanguageContext = createContext<LanguageContextValue | null>(null)

export function LanguageProvider({ children }: PropsWithChildren) {
  const [language, setLanguageState] = useState<Language>(readLanguage)
  const value = useMemo<LanguageContextValue>(() => ({
    language,
    setLanguage(next) {
      setLanguageState(next)
      localStorage.setItem(storageKey, JSON.stringify({ version: 1, language: next }))
    },
  }), [language])
  useEffect(() => { document.documentElement.lang = language }, [language])
  return <LanguageContext value={value}>{children}</LanguageContext>
}

export function useLanguage(): LanguageContextValue {
  const value = useContext(LanguageContext)
  if (!value) throw new Error('LanguageProvider is missing.')
  return value
}

export function useCopy<T>(copy: Record<Language, T>): T {
  return copy[useLanguage().language]
}

function readLanguage(): Language {
  try {
    const stored = JSON.parse(localStorage.getItem(storageKey) ?? 'null') as { version?: unknown; language?: unknown } | null
    if (stored?.version === 1 && (stored.language === 'ru' || stored.language === 'en')) return stored.language
  } catch { /* invalid local state falls back safely */ }
  return navigator.language.toLowerCase().startsWith('ru') ? 'ru' : 'en'
}
