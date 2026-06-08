import React, { createContext, useContext, useState, useCallback } from 'react';
import pl from './pl';
import en from './en';

export type Lang = 'pl' | 'en';

const dictionaries: Record<Lang, Record<string, string>> = { pl, en };

interface I18nContextValue {
  t: (key: string, params?: Record<string, string | number>) => string;
  lang: Lang;
  setLang: (l: Lang) => void;
}

const I18nContext = createContext<I18nContextValue>({
  t: (k: string) => k,
  lang: 'en',
  setLang: () => {},
});

export function useT() {
  return useContext(I18nContext);
}

interface I18nProviderProps {
  initialLang: Lang;
  children: React.ReactNode;
}

export function I18nProvider({ initialLang, children }: I18nProviderProps) {
  const [lang, setLangState] = useState<Lang>(initialLang);

  const setLang = useCallback((l: Lang) => {
    setLangState(l);
  }, []);

  const t = useCallback((key: string, params?: Record<string, string | number>): string => {
    const dict = dictionaries[lang] || dictionaries.en;
    let val = dict[key];
    if (val === undefined) {
      // Fallback to English
      val = dictionaries.en[key];
    }
    if (val === undefined) {
      return key;
    }
    if (params) {
      for (const [k, v] of Object.entries(params)) {
        val = val.replace(`{${k}}`, String(v));
      }
    }
    return val;
  }, [lang]);

  return (
    <I18nContext.Provider value={{ t, lang, setLang }}>
      {children}
    </I18nContext.Provider>
  );
}
