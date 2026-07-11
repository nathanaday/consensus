import { ref, type Ref } from 'vue'

export type Theme = 'light' | 'dark'

const theme: Ref<Theme> = ref('light')

const STORAGE_KEY = 'consensus-theme'

function apply(next: Theme) {
  theme.value = next
  document.documentElement.dataset.theme = next
}

export function initTheme(): void {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'light' || stored === 'dark') {
    apply(stored)
    return
  }
  apply(window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
}

export function useTheme(): { theme: Ref<Theme>; toggleTheme: () => void } {
  const toggleTheme = () => {
    const next: Theme = theme.value === 'light' ? 'dark' : 'light'
    localStorage.setItem(STORAGE_KEY, next)
    apply(next)
  }
  return { theme, toggleTheme }
}
