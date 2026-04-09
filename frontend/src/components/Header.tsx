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
      <div className="flex items-center gap-2">
        <h1 className="text-lg font-bold">{t('app.title')}</h1>
        <a
          href={t('dashboard.dora_link_url')}
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs text-gray-400 dark:text-gray-500 hover:text-blue-500 dark:hover:text-blue-400"
        >
          {t('dashboard.dora_link')} ↗
        </a>
      </div>
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
