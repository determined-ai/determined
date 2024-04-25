/**
 * Appends a timestamp and a short random hash to a base name to avoid naming collisions.
 * @param baseName The base name to append the timestamp and hash to.
 * @returns The base name with appended timestamp and hash.
 */
export function safeName(baseName: string): string {
  const timestamp = Date.now();
  return `${baseName}_${timestamp}_${randIdAlphanumeric()}`;
}

/**
 * Generates a four-character random hash
 * @returns Alphanumeric hash
 */
export const randIdAlphanumeric = (): string => Math.random().toString(36).substring(2, 6);

/**
 * Generates a four-character numeric hash
 * @returns Numeric hash
 */
export const randId = (): number => Math.floor(Math.random() * 10_000);
