/* eslint-disable */
/// <reference types="react-scripts" />
/// <reference path="types.ts" />

export {};

declare global {
  interface Window {
    analytics: any;
    dev: any;
  }

  interface Array<T> {
    first(): T;
    last(): T;
    random(): T;
    sortAll(compareFn: (a: T, b: T) => number): Array<T>;
  }
}

declare module global {
  namespace NodeJS {
    interface ProcessEnv {
      IS_DEV: boolean;
      VERSION: string;
      SERVER_ADDRESS?: string;
    }
  }
}
