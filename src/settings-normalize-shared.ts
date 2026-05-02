export function isPositiveNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value) && value > 0;
}

export function normalizeString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

export function normalizeBoolean(value: unknown, fallback: boolean): boolean {
  return typeof value === "boolean" ? value : fallback;
}

export function normalizeList(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.map((item) => String(item).trim()).filter(Boolean);
  }

  if (typeof value === "string") {
    return value.split(",").map((item) => item.trim()).filter(Boolean);
  }

  return [];
}

export function normalizeProviderIds(value: unknown): string[] {
  return Array.isArray(value)
    ? value.map((item) => String(item).trim()).filter(Boolean)
    : [];
}

export function normalizeInteger(value: unknown, fallback: number, fieldName: string): number {
  const nextValue = isPositiveNumber(value) ? Math.trunc(value) : fallback;
  if (!Number.isInteger(nextValue)) {
    throw new Error(`${fieldName} must be an integer.`);
  }

  return nextValue;
}

export function normalizeOptionalInteger(value: unknown, fallback: number | null, fieldName: string): number | null {
  if (value === null || value === undefined || value === "") {
    return fallback;
  }

  const parsed = typeof value === "string" ? Number(value) : value;
  if (!isPositiveNumber(parsed)) {
    throw new Error(`${fieldName} must be a positive number.`);
  }

  const nextValue = Math.trunc(parsed);
  if (!Number.isInteger(nextValue)) {
    throw new Error(`${fieldName} must be an integer.`);
  }

  return nextValue;
}
