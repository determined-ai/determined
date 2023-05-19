export type Primitive = boolean | number | string;
export type RecordKey = string | number | symbol;

export type ValueOf<T> = T[keyof T];
