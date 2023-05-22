export type Primitive = boolean | number | string;
export type RecordKey = string | number | symbol;

export type ValueOf<T> = T[keyof T];

export const Scale = {
  Linear: 'linear',
  Log: 'log',
} as const;

export type Scale = ValueOf<typeof Scale>;
