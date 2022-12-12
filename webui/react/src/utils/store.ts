// eslint-disable-next-line  @typescript-eslint/no-explicit-any
export const encodeParams = (params: { [key: string]: any }): string =>
  JSON.stringify([...Object.entries(params ?? {})].sort());
