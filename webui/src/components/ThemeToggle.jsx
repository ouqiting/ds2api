import { Moon, Sun } from 'lucide-react'
import { useEffect, useState } from 'react'

import { applyTheme, getInitialTheme, LIGHT_THEME, THEME_STORAGE_KEY, toggleTheme } from '../theme'
import { useI18n } from '../i18n'

export default function ThemeToggle({ className = '' }) {
    const { t } = useI18n()
    const [theme, setTheme] = useState(() => getInitialTheme(typeof localStorage === 'undefined' ? null : localStorage))
    const isLight = theme === LIGHT_THEME

    useEffect(() => {
        applyTheme(theme, document.documentElement)
        localStorage.setItem(THEME_STORAGE_KEY, theme)
    }, [theme])

    return (
        <button
            type="button"
            onClick={() => setTheme(toggleTheme(theme))}
            className={`p-1.5 rounded-md border border-border bg-secondary/50 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors ${className}`}
            title={isLight ? t('theme.switchToDark') : t('theme.switchToLight')}
            aria-label={isLight ? t('theme.switchToDark') : t('theme.switchToLight')}
        >
            {isLight ? <Moon className="w-3.5 h-3.5" /> : <Sun className="w-3.5 h-3.5" />}
        </button>
    )
}
