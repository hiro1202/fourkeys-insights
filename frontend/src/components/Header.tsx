import { useI18n } from '../i18n/context'
import { useEffect, useState } from 'react'

export function Header() {
  const { lang, setLang, t } = useI18n()
  const [dark, setDark] = useState(() => {
    const stored = localStorage.getItem('theme')
    if (stored) return stored === 'dark'
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  })

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark)
    localStorage.setItem('theme', dark ? 'dark' : 'light')
  }, [dark])

  return (
    <header className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-3 flex items-center justify-between">
      <h1 className="text-lg font-bold">{t('app.title')}</h1>
      <div className="flex items-center gap-3">
        <button
          onClick={() => setDark(!dark)}
          className="p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
          aria-label="Toggle dark mode"
        >
          {dark ? '☀️' : '🌙'}
        </button>
        <select
          value={lang}
          onChange={(e) => setLang(e.target.value as 'en' | 'ja')}
          className="text-sm bg-transparent border border-gray-300 dark:border-gray-600 rounded px-2 py-1"
        >
          <option value="en">EN</option>
          <option value="ja">JP</option>
        </select>
      </div>
    </header>
  )
}
