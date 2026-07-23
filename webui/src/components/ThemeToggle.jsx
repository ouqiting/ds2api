import { Moon, Sun } from 'lucide-react'
import { useEffect, useState } from 'react'

import { useI18n } from '../i18n'
import { applyTheme, getInitialTheme, LIGHT_THEME, THEME_STORAGE_KEY, toggleTheme } from '../theme'

export default function ThemeToggle({ className = '', compact = false }) {
    const { t } = useI18n()
    const [theme, setTheme] = useState(() => getInitialTheme(typeof window === 'undefined' ? undefined : window.localStorage))

    useEffect(() => {
        applyTheme(theme, document.documentElement)
        window.localStorage.setItem(THEME_STORAGE_KEY, theme)
    }, [theme])

    const nextTheme = toggleTheme(theme)
    const label = nextTheme === LIGHT_THEME ? t('theme.light') : t('theme.dark')
    const title = nextTheme === LIGHT_THEME ? t('theme.switchToLight') : t('theme.switchToDark')

    return (
        <button
            type="button"
            onClick={() => setTheme(nextTheme)}
            className={`inline-flex items-center justify-center gap-1.5 rounded-lg border border-border bg-card/80 text-xs font-medium text-muted-foreground shadow-sm backdrop-blur transition-colors hover:border-primary/40 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background ${compact ? 'h-8 w-8 px-0' : 'h-9 px-3'} ${className}`}
            title={title}
            aria-label={title}
        >
            {nextTheme === LIGHT_THEME ? <Sun className="h-3.5 w-3.5" /> : <Moon className="h-3.5 w-3.5" />}
            {!compact && <span className="hidden sm:inline">{label}</span>}
        </button>
    )
}
