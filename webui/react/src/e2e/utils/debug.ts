export function printMap<T1, T2>(map: Map<T1, T2>): string {
  return Array.from(map.entries())
    .map(([key, value]) => `${key}: ${value}`)
    .join('\n');
}
