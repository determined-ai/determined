/* eslint-disable */
/// <reference path="types.ts" />
/// <reference types="vitest/globals" />
/// <reference types="./react-svg.d.ts" />
/// <reference types="vite/client" />
import type { TestingLibraryMatchers } from '@testing-library/jest-dom/matchers';

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

declare global {
  namespace NodeJS {
    interface ProcessEnv {
      VERSION: string;
      SERVER_ADDRESS?: string;
      PUBLIC_URL: string;
    }
  }
}

declare global {
  namespace jest {
    interface Matchers<R = void>
      extends TestingLibraryMatchers<typeof expect.stringContaining, R> {}
  }
}

declare module '*.svg' {
  const ReactComponent = React.FC<React.SVGProps>;
  export default ReactComponent;
}
