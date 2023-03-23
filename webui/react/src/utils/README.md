This folder contains a number of utility types and functions for working in the codebase. To learn how to use these, read on.

# Loadable

Loadables are a simple data type to represent data that has to be loaded from the API. You might be familiar with the following pattern:

```typescript
type MyApiReturn = {
    data?: MyDataType,
    isLoading: bool,
}
const fetcher: () => MyApiReturn = ...
```

This works alright in simple cases but becomes challenging to work with when you have multiple API calls you need to combine. Typescript doesn't know about the relationship with `isLoading` and `data` and there's no distinguish between types that are loading or just optional.

To solve these we introduce a purpose-built type called `Loadable`. In simplified terms a Loadable looks like this:

```typescript
type Loadable<Data> = Loaded(data: Data) | NotLoaded

const fetcher: () => Loadable<MyDataType> = ...
```

Loadable has two constructors, `Loaded` and `NotLoaded`:

```typescript
const loadedNumber: Loadable<number> = Loaded(5);
const notLoadedNumber: Loadable<number> = NotLoaded;
```

## How to Tell if a Loadable is Loaded

There are a number of functions you can use to tell if a Loadable is loaded or not. You should only use these when you need to deal with the NotLoaded case. If you just need to change the value _inside_ the Loadable look below at "Operating on Loadables".

### match

The simplest way to tell if a Loadable is Loaded is to pass it to `Loadable.match`:

```typescript
Loadable.match(loadableNumber, {
    Loaded: (number) => <loaded codepath>,
    NotLoaded: () => <loading codepath>,
})
```

When `loadableNumber` is `Loaded(5)`, the match will execute the loaded codepath, when it is `NotLoaded` it will execute the loading codepath. This is very similar to a javascript `switch` statement. We're providing a case for each possible value of `loadableNumber`.

### getOrElse

Sometimes you just need the value inside the loadable or some default. In these cases you can use `Loadable.getOrElse`:

```typescript
const myName = Loadable.getOrElse('loading...', loadableName);
```

The value of `myName` will be `'loading...'` for as long as it takes `loadableName` to load, after which it will switch to the value of `loadableName`.

### isLoaded

Occasionally you just need to know if a Loadable is loaded but you don't care about the value inside, for those cases you can use `Loadable.isLoaded`. This simply returns `true` if the Loadable is `Loaded` and `false` if it is `NotLoaded`.

## Operating on Loadables

When programming with API values, you often don't care (yet) whether the value is loaded or not. You might need to create a derived value or combine values from multiple APIs but leave the loading states to another component. For these cases there are a number of functions at your disposal.

### map

`map` will take any function and apply it to the value inside the `Loadable` if it is loaded.

```typescript
Loadable.map(addOne, Loaded(1)); // Loaded(2)
```

### forEach

`forEach` can be used to run a side-effecting function on a `Loadable`. The side-effect will run only if the `Loadable` is loaded, otherwise this is a no-op.

## Combining Loadables

Oftentimes we care about multiple values from the API. For example we may want a page to display a spinner until worth a workspace and its projects finish loading or we may need to derive one loadable value from another.

### all

`all` takes a tuple of `Loadable`s and returns a `Loadable` of a tuple. So a `[Loadable<string>, Loadable<number>]` becomes a `Loadable<[string, number]>`.

### flatMap

Sometimes two loadables you need to combine don't exist at the same time, that is one relies on the _value_ of the other. In these cases `flatMap` provides a more flexible way of combining `Loadable`s.

```typescript
const getTags: (ids: Array<Id<Tag>>) => Loadable<Array<Tag>> = ...
const myObject: Loadable<SomeObject> = ...

const myTags: Loadable<Array<Tag>> = Loadable.flatMap(myObject, o => getTags(o.tags))
```

## Use With React.Suspense

It's not always prudent to deal with the loading state in the same place you need the value. For those situations React.Suspense provides a great solution. To throw to a Suspense boundary simply call `Loadable.waitFor` on the `Loadable` value and then use it as a normal value.

## Designing APIs with Loadable

While we mainly think of APIs _returning_ `Loadable`s, sometimes APIs need to _accept_ `Loadable`s as well. In this case it's often prudent to accept both a `Loadable` and a regular value to cover all use-cases and avoid unnecessary wrapping, you can do this with `isLoadable`.

```typescript
interface Props {
    value: string | Loadable<string>
}

const MyComponent = ({ value }: Props) => {
    const value_ = Loadable.isLoadable(value) ? value : Loaded(value);

    return Loadable.match(value_, ...)
}

```

Oftentimes we want _React hooks_ to take a `Loadable` value, in these cases make sure to call a no-op version of the hook in the `NotLoaded` case to preserve the hook identities.

# Observables

See [micro-observables](https://github.com/betomorrow/micro-observables).
