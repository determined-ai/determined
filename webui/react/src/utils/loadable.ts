/* eslint-disable @typescript-eslint/no-explicit-any */
export type Loadable<T> =
  | {
      _tag: 'Loaded';
      data: T;
    }
  | {
      _tag: 'NotLoaded';
    }
  | {
      _tag: 'NotFound';
    };

const exhaustive = (v: never): never => v;

const Loaded = <T>(data: T): Loadable<T> => ({ _tag: 'Loaded', data });
const NotLoaded: Loadable<never> = { _tag: 'NotLoaded' };
const NotFound: Loadable<never> = { _tag: 'NotFound' };

const map = <T, U>(l: Loadable<T>, fn: (_: T) => U): Loadable<U> => {
  switch (l._tag) {
    case 'Loaded':
      return Loaded(fn(l.data));
    case 'NotLoaded':
      return NotLoaded;
    case 'NotFound':
      return NotFound;
    default:
      return exhaustive(l);
  }
};

const flatMap = <T, U>(l: Loadable<T>, fn: (_: T) => Loadable<U>): Loadable<U> => {
  const res = map(l, fn);
  if (res._tag === 'Loaded') {
    return res.data;
  }
  return res;
};

const forEach = <T, U>(l: Loadable<T>, fn: (_: T) => U): void => {
  switch (l._tag) {
    case 'Loaded': {
      fn(l.data);
      return;
    }
    case 'NotLoaded':
      return;
    case 'NotFound':
      return;
    default:
      exhaustive(l);
  }
};

const getOrElse = <T>(def: T, l: Loadable<T>): T => {
  switch (l._tag) {
    case 'Loaded':
      return l.data;
    case 'NotLoaded':
      return def;
    case 'NotFound':
      return def;
    default:
      return exhaustive(l);
  }
};

type MatchArgs<T, U> =
  | {
      Loaded: (data: T) => U;
      NotFound: () => U;
      NotLoaded: () => U;
    }
  | {
      Loaded: (data: T) => U;
      _: () => U;
    }
  | {
      NotFound: () => U;
      _: () => U;
    }
  | {
      NotLoaded: () => U;
      _: () => U;
    };
const match = <T, U>(l: Loadable<T>, cases: MatchArgs<T, U>): U => {
  switch (l._tag) {
    case 'Loaded':
      return 'Loaded' in cases ? cases.Loaded(l.data) : cases._();
    case 'NotLoaded':
      return 'NotLoaded' in cases ? cases.NotLoaded() : cases._();
    case 'NotFound':
      return 'NotFound' in cases ? cases.NotFound() : cases._();
    default:
      return exhaustive(l);
  }
};

const quickMatch = <T, U>(l: Loadable<T>, def: U, f: (data: T) => U): U => {
  switch (l._tag) {
    case 'Loaded':
      return f(l.data);
    case 'NotLoaded':
      return def;
    case 'NotFound':
      return def;
    default:
      return exhaustive(l);
  }
};

const isLoadable = <T, Z>(l: Loadable<T> | Z): l is Loadable<T> => {
  return ['Loaded', 'NotLoaded', 'NotFound'].includes((l as Loadable<T>)?._tag);
};

/**
 * Groups up all passed Loadables. NotFound takes priority over
 * NotLoaded so all([NotLoaded, NotFound, Loaded(4)]) returns NotFound
 */
function all<A>(ls: [Loadable<A>]): Loadable<[A]>;
function all<A, B>(ls: [Loadable<A>, Loadable<B>]): Loadable<[A, B]>;
function all<A, B, C>(ls: [Loadable<A>, Loadable<B>, Loadable<C>]): Loadable<[A, B, C]>;
function all<A, B, C, D>(
  ls: [Loadable<A>, Loadable<B>, Loadable<C>, Loadable<D>],
): Loadable<[A, B, C, D]>;
function all<A, B, C, D, E>(
  ls: [Loadable<A>, Loadable<B>, Loadable<C>, Loadable<D>, Loadable<E>],
): Loadable<[A, B, C, D, E]>;
function all(ls: Array<Loadable<any>>): Loadable<Array<any>> {
  const res: any[] = [];
  for (const l of ls) {
    if (l._tag === 'NotFound') {
      return NotFound;
    }
    if (l._tag === 'NotLoaded') {
      return NotLoaded;
    }
    res.push(l.data);
  }
  return Loaded(res);
}

// exported immediately because of name collision
export const Loadable = { all, flatMap, forEach, getOrElse, isLoadable, map, match, quickMatch };

export { Loaded, NotLoaded, NotFound };
