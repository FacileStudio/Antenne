import { rememberLocale, resolveClientLocale } from '@facile/client/locale';
import { createTranslator, LOCALES, resolveLocale, type Locale } from '@facile/muse';
import { catalogs } from './messages';

const session = $state<{ locale: string | null }>({ locale: null });
let active: () => Locale = () => 'en';

/** Records the locale the signed-in person has on their Registre profile. */
export function setSessionLocale(locale: string | null) {
	session.locale = locale;
}

/** The locale source for `provideLocale`: the URL, the session, the picked cookie, then the browser. Null when nothing matched. */
export function localeSource(): string | null {
	return resolveClientLocale({ supported: LOCALES, profile: session.locale }).hint;
}

export { rememberLocale };

/** Points the translator and the Intl helpers at the locale `provideLocale` resolved. */
export function bindLocale(source: () => Locale) {
	active = source;
}

/** The locale currently in force. */
export function locale(): Locale {
	return resolveLocale(active());
}

/** Translates an app message in the current locale. */
export const t = createTranslator(catalogs, () => active());
