declare namespace NodeJS {
  export interface ProcessEnv {
    VERSION: string;
  }
}

export declare global {
  interface Window {
    /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
    analytics: any;
  }
}
