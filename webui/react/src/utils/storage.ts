export interface Store {
  clear(): void;
  getItem(key: string): string | null;
  keys(): string[];
  removeItem(key: string): void;
  setItem(key: string, value: string): void;
}

interface StorageOptions {
  basePath?: string;
  delimiter?: string;
  store: Store;
}

export class MemoryStore implements Store {
  private store: Record<string, string>;

  constructor() {
    this.store = {};
  }

  clear(): void {
    this.store = {};
  }

  getItem(key: string): string | null {
    if (key in this.store) return this.store[key];
    return null;
  }

  removeItem(key: string): void {
    delete this.store[key];
  }

  setItem(key: string, value: string): void {
    this.store[key] = value;
  }

  keys(): string[] {
    return Object.keys(this.store);
  }
}

export class Storage {
  private delimiter: string;
  private pathKeys: string[];
  private store: Store;

  constructor(options: StorageOptions) {
    this.delimiter = options.delimiter || '/';
    this.pathKeys = this.parsePath(options.basePath || '', this.delimiter);
    this.store = options.store;
  }

  clear(): void {
    this.store.clear();
  }

  get<T>(key: string): T | null {
    const path = this.computeKey(key);
    const item = this.store.getItem(path);
    if (item !== null) return JSON.parse(item);
    return null;
  }

  getWithDefault<T>(key: string, defaultValue: T): T {
    const value = this.get<T>(key);
    return value !== null ? value : defaultValue;
  }

  remove(key: string): void {
    const path = this.computeKey(key);
    this.store.removeItem(path);
  }

  set<T>(key: string, value: T): void {
    if (value == null) throw new Error('Cannot set to a null or undefined value.');
    if (value instanceof Set) throw new Error('Convert the value to an Array before setting it.');
    const path = this.computeKey(key);
    const item = JSON.stringify(value);
    this.store.setItem(path, item);
  }

  keys(): string[] {
    const prefix = this.pathKeys.join(this.delimiter) + this.delimiter;
    return this.store.keys()
      .filter(key => key.startsWith(prefix))
      .map(key => key.replace(prefix, ''));
  }

  toString(): string {
    const inMemoryRepr = this.keys().reduce((acc: Record<string, unknown>, cur: string) => {
      cur;
      acc[cur] = this.get(cur);
      return acc;
    }, {});

    return JSON.stringify(inMemoryRepr);
  }

  fromString(marshalled: string): void {
    const inMemoryRepr = JSON.parse(marshalled);
    for (const key in inMemoryRepr) {
      this.set(key, inMemoryRepr[key]);
    }
  }

  reset(): void {
    this.keys().forEach(key => this.remove(key));
  }

  fork(basePath: string): Storage {
    basePath = [ ...this.pathKeys, basePath ].join(this.delimiter);
    return new Storage({ basePath, delimiter: this.delimiter, store: this.store });
  }
  private computeKey(key: string): string {
    return [ ...this.pathKeys, key ].join(this.delimiter);
  }

  private parsePath (path: string, delimiter: string): string[] {
    return path.split(delimiter).filter(key => key !== '');
  }

}
