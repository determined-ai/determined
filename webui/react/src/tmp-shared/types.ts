export type Primitive = boolean | number | string;
export type RecordKey = string | number | symbol;
export type UnknownRecord = Record<RecordKey, unknown>;
export type NullOrUndefined<T = undefined> = T | null | undefined;
export type Point = { x: number; y: number };
export type Range<T = Primitive> = [ T, T ];
export type Eventually<T> = T | Promise<T>;
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export type RawJson = Record<string, any>;

export interface Pagination {
  limit: number;
  offset: number;
}
