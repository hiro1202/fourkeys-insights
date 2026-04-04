import { createContext, useContext, useState, useCallback, ReactNode } from 'react'
import en from './en.json'
import ja from './ja.json'

type Lang = 'en' | 'ja'
type Translations = Record<string, string>

const translations: Record<Lang, Translations> = { en, ja }

interface I18nContextType {
  lang: Lang
  setLang: (lang: Lang) => void
  t: (key: string, params?: Record<string, string | number>) => string
}

const I18nContext = createContext<I18nContextType | null>(null)

function getInitialLang(): Lang {
  const stored = localStorage.getItem('lang')
  if (stored === 'ja' || stored === 'en') return stored
  return navigator.language.startsWith('ja') ? 'ja' : 'en'
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(getInitialLang)

  const setLang = useCallback((newLang: Lang) => {
    setLangState(newLang)
    localStorage.setItem('lang', newLang)
  }, [])

  const t = useCallback((key: string, params?: Record<string, string | number>) => {
    let text = translations[lang][key] ?? translations.en[key] ?? key
    if (params) {
      for (const [k, v] of Object.entries(params)) {
        text = text.replace(`{${k}}`, String(v))
      }
    }
    return text
  }, [lang])

  return (
    <I18nContext.Provider value={{ lang, setLang, t }}>
      {children}
    </I18nContext.Provider>
  )
}

export function useI18n() {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error('useI18n must be used within I18nProvider')
  return ctx
}
