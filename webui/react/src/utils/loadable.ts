export type Loadable<T> =
  | {
    _tag: 'Loaded';
    data: T;
  }
  | {
    _tag: 'NotLoaded';
  };

const exhaustive = (v: never): never => v;

const Loaded = <T>(data: T): Loadable<T> => ({ _tag: 'Loaded', data });
const NotLoaded: Loadable<never> = { _tag: 'NotLoaded' };

/**
 * The map() function creates a new Loadable with the result of calling
 * the provided function on the contained value in the passed Loadable.
 *
 * If the passed Loadable is NotLoaded then the return value is NotLoaded
 */
const map = <T, U>(l: Loadable<T>, fn: (_: T) => U): Loadable<U> => {
  switch (l._tag) {
    case 'Loaded':
      return Loaded(fn(l.data));
    case 'NotLoaded':
      return NotLoaded;
    default:
      return exhaustive(l);
  }
};

/**
 * The flatMap() function creates a new Loadable with the result of calling
 * the provided function on the contained value in the passed Loadable and then
 * flattening the result.
 *
 * If any of the passed or returned Loadables is NotLoaded, the result is NotLoaded.
 */
const flatMap = <T, U>(l: Loadable<T>, fn: (_: T) => Loadable<U>): Loadable<U> => {
  const res = map(l, fn);
  if (res._tag === 'Loaded') {
    return res.data;
  }
  return res;
};

/**
 * Performs a side-effecting function if the passed Loadable is Loaded.
 */
const forEach = <T, U>(l: Loadable<T>, fn: (_: T) => U): void => {
  switch (l._tag) {
    case 'Loaded': {
      fn(l.data);
      return;
    }
    case 'NotLoaded':
      return;
    default:
      exhaustive(l);
  }
};

/**
 * If the passed Loadable is Loaded this returns the data, otherwise
 * it returns the default value.
 */
const getOrElse = <T>(def: T, l: Loadable<T>): T => {
  switch (l._tag) {
    case 'Loaded':
      return l.data;
    case 'NotLoaded':
      return def;
    default:
      return exhaustive(l);
  }
};

type MatchArgs<T, U> =
  | {
    Loaded: (data: T) => U;
    NotLoaded: () => U;
  }
  | {
    Loaded: (data: T) => U;
    _: () => U;
  }
  | {
    NotLoaded: () => U;
    _: () => U;
  };
/**
 * Allows you to match out the cases in the Loadable with named
 * arguments.
 */
const match = <T, U>(l: Loadable<T>, cases: MatchArgs<T, U>): U => {
  switch (l._tag) {
    case 'Loaded':
      return 'Loaded' in cases ? cases.Loaded(l.data) : cases._();
    case 'NotLoaded':
      return 'NotLoaded' in cases ? cases.NotLoaded() : cases._();
    default:
      return exhaustive(l);
  }
};

/** Like `match` but without argument names */
const quickMatch = <T, U>(l: Loadable<T>, def: U, f: (data: T) => U): U => {
  switch (l._tag) {
    case 'Loaded':
      return f(l.data);
    case 'NotLoaded':
      return def;
    default:
      return exhaustive(l);
  }
};

/** Returns true if the passed object is a Loadable */
const isLoadable = <T, Z>(l: Loadable<T> | Z): l is Loadable<T> => {
  return ['Loaded', 'NotLoaded', 'NotFound'].includes((l as Loadable<T>)?._tag);
};

const isLoading = <T>(l: Loadable<T>): l is { _tag: 'NotLoaded' } => {
  return l === NotLoaded;
};

const isLoaded = <T>(l: Loadable<T>): l is { _tag: 'Loaded'; data: T } => {
  return l !== NotLoaded;
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
function all(ls: Array<Loadable<unknown>>): Loadable<Array<unknown>> {
  const res: unknown[] = [];
  for (const l of ls) {
    if (l._tag === 'NotLoaded') {
      return NotLoaded;
    }
    res.push(l.data);
  }
  return Loaded(res);
}

/** Allows you to use Loadables with React's Suspense component */
const waitFor = <T>(l: Loadable<T>): T => {
  switch (l._tag) {
    case 'Loaded':
      return l.data;
    case 'NotLoaded':
      throw Promise.resolve(undefined);
    default:
      return exhaustive(l);
  }
};

// exported immediately because of name collision
export const Loadable = {
  all,
  flatMap,
  forEach,
  getOrElse,
  isLoadable,
  isLoaded,
  isLoading,
  map,
  match,
  quickMatch,
  waitFor,
};

export { Loaded, NotLoaded };
