interface StorageOptions {
  basePath?: string;
  delimiter?: string;
  store: Storage;
}

export class MemoryStore implements Storage {
  // MemoryStore is used only in tests, and key/length are not used,
  // only added for compatibility with localStorage type.
  length: 0;
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

  key(index: number): string {
    return Object.keys(this.store)[index];
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

export class StorageManager {
  private delimiter: string;
  private pathKeys: string[];
  private store: Storage;

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

  remove(key: string, storagePath?: string): void {
    if (storagePath && this.getStoragePath() !== storagePath) return;
    const path = this.computeKey(key);
    this.store.removeItem(path);
  }

  set<T>(key: string, value: T, storagePath?: string): void {
    if (value == null) throw new Error('Cannot set to a null or undefined value.');
    if (value instanceof Set) throw new Error('Convert the value to an Array before setting it.');
    if (storagePath && this.getStoragePath() !== storagePath) return;
    const path = this.computeKey(key);
    const item = JSON.stringify(value);
    this.store.setItem(path, item);
  }

  keys(): string[] {
    const prefix = this.pathKeys.length !== 0 ? [...this.pathKeys, ''].join(this.delimiter) : '';
    return this.store
      .keys()
      .filter((key: string) => key.startsWith(prefix))
      .map((key: string) => key.replace(prefix, ''));
  }

  toString(): string {
    const inMemoryRecord = this.keys().reduce((acc, key) => {
      acc[key] = this.get(key);
      return acc;
    }, {} as Record<string, unknown>);

    return JSON.stringify(inMemoryRecord);
  }

  fromString(marshalled: string): void {
    const inMemoryRecord = JSON.parse(marshalled);
    for (const key in inMemoryRecord) {
      this.set(key, inMemoryRecord[key]);
    }
  }

  fork(basePath: string): StorageManager {
    basePath = [...this.pathKeys, basePath].join(this.delimiter);
    return new StorageManager({ basePath, delimiter: this.delimiter, store: this.store });
  }

  reset(): void {
    this.keys().forEach((key) => this.remove(key));
  }

  getStoragePath(): string {
    return this.computeKey('').slice(0, -1); // because the last char is the delimiter
  }

  private computeKey(key: string): string {
    return [...this.pathKeys, key].join(this.delimiter);
  }

  private parsePath(path: string, delimiter: string): string[] {
    return path.split(delimiter).filter((key) => key !== '');
  }
}
