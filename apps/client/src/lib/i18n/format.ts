import { locale } from './index.svelte';

/** Formats a date or instant with `Intl.DateTimeFormat` in the current locale. */
export function formatDate(value: Date | string | number, options: Intl.DateTimeFormatOptions = {}): string {
	return new Intl.DateTimeFormat(locale(), options).format(new Date(value));
}

/** Formats a number with `Intl.NumberFormat` in the current locale. */
export function formatNumber(value: number, options: Intl.NumberFormatOptions = {}): string {
	return new Intl.NumberFormat(locale(), options).format(value);
}

/** Formats a quantity with its unit, in the short style of the current locale. */
export function formatUnit(value: number, unit: 'second' | 'minute' | 'hour' | 'day'): string {
	return new Intl.NumberFormat(locale(), { style: 'unit', unit, unitDisplay: 'narrow' }).format(value);
}
