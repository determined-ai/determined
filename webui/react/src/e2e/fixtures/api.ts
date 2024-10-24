import { BaseAPI } from 'services/api-ts-sdk';

export type ApiArgsFixture = ConstructorParameters<typeof BaseAPI>;
export type ApiConstructor<T> = new (...args: ConstructorParameters<typeof BaseAPI>) => T;

// We need to infer the type of the resulting class here because it's a mixin
// eslint-disable-next-line @typescript-eslint/explicit-module-boundary-types
export const apiFixture = <T>(api: ApiConstructor<T>) => {
  return class {
    protected readonly api: T;
    constructor(protected apiArgs: ApiArgsFixture) {
      this.api = new api(...apiArgs);
    }
    protected get token() {
      return this.apiArgs[0]?.apiKey;
    }
    protected get baseUrl() {
      return this.apiArgs[1];
    }
  };
};
