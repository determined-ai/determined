/* eslint-disable @typescript-eslint/no-non-null-assertion */
const exhaustive = (v: never): never => v;

type MatchArgs<T, U> =
  | {
      Loaded: (data: T) => U;
      NotLoaded: () => U;
      Failed: (e: Error) => U;
    }
  | {
      Loaded: (data: T) => U;
      NotLoaded: () => U;
      _: () => U;
    }
  | {
      Loaded: (data: T) => U;
      Failed: (e: Error) => U;
      _: () => U;
    }
  | {
      NotLoaded: () => U;
      Failed: (e: Error) => U;
      _: () => U;
    }
  | {
      Loaded: (data: T) => U;
      _: () => U;
    }
  | {
      NotLoaded: () => U;
      _: () => U;
    }
  | {
      Failed: (e: Error) => U;
      _: () => U;
    };

class Loadable_<T> {
  _tag: 'Loaded' | 'NotLoaded' | 'Failed';
  data: T | undefined;
  error: Error | undefined;

  constructor(
    tag: 'Loaded' | 'NotLoaded' | 'Failed',
    data: T | undefined,
    error: Error | undefined = undefined,
  ) {
    this._tag = tag;
    this.data = data;
    this.error = error;
  }

  /**
   * The map() function creates a new Loadable with the result of calling
   * the provided function on the contained value in the passed Loadable.
   *
   * If the passed Loadable is NotLoaded then the return value is NotLoaded
   */
  map<U>(fn: (t: T) => U): Loadable<U> {
    switch (this._tag) {
      case 'Loaded':
        return new Loadable_('Loaded', fn(this.data!)) as Loadable<U>;
      case 'NotLoaded':
        return NotLoaded as Loadable<U>;
      case 'Failed':
        return new Loadable_<U>('Failed', undefined, this.error) as Loadable<U>;
      default:
        return exhaustive(this._tag);
    }
  }
  static map<T, U>(l: Loadable<T>, fn: (_: T) => U): Loadable<U> {
    return l.map(fn);
  }

  /**
   * The flatMap() function creates a new Loadable with the result of calling
   * the provided function on the contained value in the passed Loadable and then
   * flattening the result.
   *
   * If any of the passed or returned Loadables is NotLoaded, the result is NotLoaded.
   */
  flatMap<U>(fn: (_: T) => Loadable<U>): Loadable<U> {
    switch (this._tag) {
      case 'Loaded':
        return fn(this.data!) as Loadable<U>;
      case 'NotLoaded':
        return NotLoaded as Loadable<U>;
      case 'Failed':
        return new Loadable_<U>('Failed', undefined, this.error) as Loadable<U>;
      default:
        return exhaustive(this._tag);
    }
  }
  static flatMap<T, U>(l: Loadable<T>, fn: (_: T) => Loadable<U>): Loadable<U> {
    return l.flatMap(fn);
  }

  /**
   * Performs a side-effecting function if the passed Loadable is Loaded.
   */
  forEach<U>(fn: (_: T) => U): void {
    switch (this._tag) {
      case 'Loaded': {
        fn(this.data!);
        return;
      }
      case 'NotLoaded':
        return;
      case 'Failed':
        return;
      default:
        exhaustive(this._tag);
    }
  }
  static forEach<T, U>(l: Loadable<T>, fn: (_: T) => U): void {
    return l.forEach(fn);
  }

  /**
   * If the passed Loadable is Loaded this returns the data, otherwise
   * it returns the default value.
   */
  getOrElse(def: T): T {
    switch (this._tag) {
      case 'Loaded':
        return this.data!;
      case 'NotLoaded':
        return def;
      case 'Failed':
        return def;
      default:
        return exhaustive(this._tag);
    }
  }
  static getOrElse<T>(def: T, l: Loadable<T>): T {
    return l.getOrElse(def);
  }

  /**
   * Allows you to match out the cases in the Loadable with named
   * arguments.
   */
  match<U>(cases: MatchArgs<T, U>): U {
    switch (this._tag) {
      case 'Loaded':
        return 'Loaded' in cases ? cases.Loaded(this.data!) : cases._();
      case 'NotLoaded':
        return 'NotLoaded' in cases ? cases.NotLoaded() : cases._();
      case 'Failed':
        return 'Failed' in cases ? cases.Failed(this.error!) : cases._();
      default:
        return exhaustive(this._tag);
    }
  }
  static match<T, U>(l: Loadable<T>, cases: MatchArgs<T, U>): U {
    return l.match(cases);
  }

  /** Like `match` but without argument names */
  quickMatch<U>(nl: U, fd: U, f: (data: T) => U): U {
    switch (this._tag) {
      case 'Loaded':
        return f(this.data!);
      case 'NotLoaded':
        return nl;
      case 'Failed':
        return fd;
      default:
        return exhaustive(this._tag);
    }
  }
  static quickMatch<T, U>(l: Loadable<T>, nl: U, fd: U, f: (data: T) => U): U {
    return l.quickMatch(nl, fd, f);
  }

  /**
   * Groups up all passed Loadables. Failed takes priority over
   * NotLoaded so all([NotLoaded, Failed, Loaded(4)]) returns Failed
   */
  static all<A>(ls: [Loadable<A>]): Loadable<[A]>;
  static all<A, B>(ls: [Loadable<A>, Loadable<B>]): Loadable<[A, B]>;
  static all<A, B, C>(ls: [Loadable<A>, Loadable<B>, Loadable<C>]): Loadable<[A, B, C]>;
  static all<A, B, C, D>(
    ls: [Loadable<A>, Loadable<B>, Loadable<C>, Loadable<D>],
  ): Loadable<[A, B, C, D]>;
  static all<A, B, C, D, E>(
    ls: [Loadable<A>, Loadable<B>, Loadable<C>, Loadable<D>, Loadable<E>],
  ): Loadable<[A, B, C, D, E]>;
  static all<T>(ls: Array<Loadable<T>>): Loadable<Array<T>>;
  static all(ls: Array<Loadable<unknown>>): Loadable<Array<unknown>> {
    const res: unknown[] = [];
    let isLoading = false;
    for (const l of ls) {
      if (l._tag === 'NotLoaded') {
        isLoading = true;
      } else if (l._tag === 'Failed') {
        return new Loadable_<unknown[]>('Failed', undefined, l.error) as Loadable<unknown[]>;
      } else {
        res.push(l.data);
      }
    }
    if (isLoading) {
      return NotLoaded as Loadable<unknown[]>;
    }
    return new Loadable_('Loaded', res) as Loadable<unknown[]>;
  }

  /**
   * Filters an array of Loadables to remove NotLoaded values and returns array of data.
   * Can also optionally accept a conditional function.
   */
  static filterNotLoaded<T>(
    a: Array<Loadable<T>>,
    conditionFn: (d: T, i?: number) => boolean = () => true,
  ): Array<T> {
    return a.flatMap((l) => (l.isLoaded ? (conditionFn(l.data) ? [l.data] : []) : []));
  }

  /** Allows you to use Loadables with React's Suspense component */
  waitFor(): T {
    switch (this._tag) {
      case 'Loaded':
        return this.data!;
      case 'NotLoaded':
        throw Promise.resolve(undefined);
      case 'Failed':
        throw this.error;
      default:
        return exhaustive(this._tag);
    }
  }
  static waitFor<T>(l: Loadable<T>): T {
    return l.waitFor();
  }
  get isLoaded(): boolean {
    return this._tag === 'Loaded';
  }
  static isLoaded<T>(
    l: Loadable<T>,
  ): l is { _tag: 'Loaded'; data: T; isLoaded: true; isNotLoaded: false; isFailed: false } & Omit<
    Loadable_<T>,
    'isLoaded' | 'data'
  > {
    return l.isLoaded;
  }
  get isNotLoaded(): boolean {
    return this._tag === 'NotLoaded';
  }
  static isNotLoaded<T>(
    l: Loadable<T>,
  ): l is { _tag: 'NotLoaded'; isLoaded: false; isNotLoaded: true; isFailed: false } & Omit<
    Loadable_<T>,
    'isNotLoaded' | 'data'
  > {
    return l.isNotLoaded;
  }
  get isFailed(): boolean {
    return this._tag === 'Failed';
  }
  static isFailed<T>(
    l: Loadable<T>,
  ): l is { _tag: 'Failed'; isLoaded: false; isNotLoaded: false; isFailed: true } & Omit<
    Loadable_<T>,
    'isFailed' | 'data'
  > {
    return l.isFailed;
  }

  /** Returns true if the passed object is a Loadable */
  static isLoadable<T, Z>(l: Loadable<T> | Z): l is Loadable<T> {
    return ['Loaded', 'NotLoaded', 'Failed'].includes((l as Loadable<T>)?._tag);
  }

  /** If passed a Loadable, returns unchanged. Otherwise, wraps argument in Loaded */
  static ensureLoadable<T>(l: Loadable<T> | T): Loadable<T> {
    return this.isLoadable(l) ? l : Loaded(l);
  }
}

export type Loadable<T> =
  | ({
      _tag: 'Loaded';
      data: T;
      isLoaded: true;
      isNotLoaded: false;
      isFailed: false;
    } & Omit<Loadable_<T>, '_tag' | 'isLoaded' | 'isNotLoaded' | 'isFailed' | 'data'>)
  | ({
      _tag: 'NotLoaded';
      isLoaded: false;
      isNotLoaded: true;
      isFailed: false;
    } & Omit<Loadable_<T>, '_tag' | 'isLoaded' | 'isNotLoaded' | 'isFailed' | 'data'>)
  | ({
      _tag: 'Failed';
      isLoaded: false;
      isNotLoaded: false;
      isFailed: true;
    } & Omit<Loadable_<T>, '_tag' | 'isLoaded' | 'isNotLoaded' | 'isFailed' | 'data'>);

// There's no real way to add methods to a union type in typescript except for with Proxies
// but Proxies don't handle generics correctly. We have to "lie" to typescript here to convince
// it that our class is a union type. It's also impossible to write custom guard types
// as methods on a class so we have to lie to it about the return type of all of our guard methods.
const Loaded = <T>(data: T): Loadable<T> => new Loadable_('Loaded', data) as Loadable<T>;
const NotLoaded: Loadable<never> = new Loadable_('NotLoaded', undefined) as Loadable<never>;
const Failed = <T>(error: Error): Loadable<T> =>
  new Loadable_<T>('Failed', undefined, error) as Loadable<T>;

export const Loadable = Loadable_;

export { Loaded, NotLoaded, Failed };
