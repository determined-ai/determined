/* eslint-disable */
/// <reference types="react-scripts" />
/// <reference path="types.ts" />

declare namespace NodeJS {
  export interface ProcessEnv {
    VERSION: string;
  }
}

export declare global {
  /* eslint-disable @typescript-eslint/no-explicit-any */
  interface Window { dev: any }
}

