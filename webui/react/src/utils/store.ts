export const encodeParams = (params: { [key: string]: any }): string =>
  JSON.stringify([...Object.entries(params ?? {})].sort());
