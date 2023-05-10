/* eslint-disable  @typescript-eslint/no-explicit-any */
type ItStandin = (name: string, fn: (...args: any) => any, timeout?: number | undefined) => void;

export const quarantinedIt = (): ItStandin => {
  return process.env.QUARANTINED ? it.skip : it;
};
