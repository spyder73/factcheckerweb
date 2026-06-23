// i18n bootstrap. EN is fully populated (the canonical strings live there);
// DE/ES/FR are stub translations marked TODO so a real translator can polish
// without touching component code.
//
// The whole resource bundle is statically imported so production builds
// don't dynamically load JSON. Acceptable for the ~10KB total we have here.

import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'

import en from './en.json'
import de from './de.json'
import es from './es.json'
import fr from './fr.json'

export const SUPPORTED_LANGS = ['en', 'de', 'es', 'fr'] as const
export type Lang = (typeof SUPPORTED_LANGS)[number]
const STORAGE_KEY = 'alethea:lang'

function detectLang(): Lang {
  if (typeof window === 'undefined') return 'en'
  const stored = window.localStorage.getItem(STORAGE_KEY) as Lang | null
  if (stored && (SUPPORTED_LANGS as readonly string[]).includes(stored)) return stored
  const nav = (window.navigator.language || 'en').toLowerCase().slice(0, 2)
  if ((SUPPORTED_LANGS as readonly string[]).includes(nav)) return nav as Lang
  return 'en'
}

void i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    de: { translation: de },
    es: { translation: es },
    fr: { translation: fr },
  },
  lng: detectLang(),
  fallbackLng: 'en',
  interpolation: { escapeValue: false }, // React already escapes
})

export function setLanguage(l: Lang): void {
  void i18n.changeLanguage(l)
  window.localStorage.setItem(STORAGE_KEY, l)
}

export default i18n
