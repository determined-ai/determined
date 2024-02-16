import { isLeft, isRight } from 'fp-ts/Either';
import * as t from 'io-ts';
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { Json, JsonArray } from 'types';
import { getProps } from 'utils/asValueObject';
import { deepObservable, useObservable } from 'utils/observable';

import { UserSettings } from './useSettingsProvider';

const stringToPrimitive = <T extends Exclude<Json, JsonArray>, U extends t.Type<T, T, unknown>>(
  str: string,
  codec: U,
): t.Validation<T> => {
  let primitive;
  if (codec instanceof t.UnionType) {
    const validations: t.Validation<T>[] = (codec.types as t.Any[]).map((c) =>
      stringToPrimitive(str, c),
    );
    return validations.find(isRight) || validations[0];
  }
  if (codec instanceof t.AnyType) {
    primitive = JSON.parse(str);
  } else if (codec.is(0)) {
    primitive = +str;
  } else if (codec.is('')) {
    primitive = str;
  } else if (codec.is(false)) {
    primitive = str === 'true';
  } else {
    primitive = JSON.parse(str);
  }
  return codec.decode(primitive);
};

/**
 * read/write typesafe params from/to the url
 */
export const useTypedParams = <T extends t.HasProps>(
  type: T,
  defaultParams: t.TypeOf<T>,
): { params: t.TypeOf<T>; updateParams: (p: Partial<t.TypeOf<T>>) => void } => {
  const { querySettings } = useContext(UserSettings);
  const typeRef = useRef(type);
  const [, setSearchParams] = useSearchParams();

  const props = useMemo(() => getProps(typeRef.current), []);

  const [paramObservable] = useState(() => {
    const rawParams = Object.entries(props).reduce(
      (acc, [key, codec]) => {
        const param: string[] = querySettings.getAll(key);
        let value;
        if (codec instanceof t.ArrayType && param.length > 0) {
          const paramList = param.map((p) => stringToPrimitive(p, codec.type));
          if (paramList.every((p) => isRight(p))) {
            value = codec.decode(paramList.filter(isRight).map((p) => p.right));
          } else {
            value = paramList.find((p) => !isRight(p));
          }
        } else if (param.length > 0) {
          value = stringToPrimitive(param[0], codec);
        }
        if (value && isRight(value)) {
          // technically a side effect, but should be okay
          querySettings.delete(key);
          acc[key] = value.right;
        }
        return acc;
      },
      {} as Record<string, unknown>,
    );

    const finalObject = {
      ...defaultParams,
      ...rawParams,
    };
    // shouldn't return null, but might!
    const validation = typeRef.current.decode(finalObject);
    if (isLeft(validation)) {
      throw new Error('unable to parse type from params and defaults');
    }
    return deepObservable(validation.right);
  });
  const params = useObservable(paramObservable);

  const updateParams = useCallback(
    (partial: Partial<t.TypeOf<T>>) => {
      paramObservable.update((p) => ({ ...p, ...partial }));
    },
    [paramObservable],
  );

  useEffect(() => {
    return paramObservable.subscribe((params) => {
      setSearchParams(
        (oldSearchParams) => {
          const searchParams = new URLSearchParams([...oldSearchParams.entries()]);
          Object.keys(props).forEach((key) => {
            searchParams.delete(key);
            // skip setting searchparam if it's missing on purpose
            if (!(key in params) || params[key] === undefined || params[key] === null) {
              return;
            }
            const valueToString = (v: unknown) =>
              typeof v === 'string' ? v.toString() : JSON.stringify(v);
            const value = params[key];
            if (Array.isArray(value)) {
              const [head, ...rest] = value;
              searchParams.set(key, valueToString(head));
              rest.forEach((v) => {
                searchParams.append(key, valueToString(v));
              });
            } else {
              searchParams.set(key, valueToString(value));
            }
          });
          searchParams.sort();
          return searchParams.toString() !== oldSearchParams.toString()
            ? searchParams
            : oldSearchParams;
        },
        { replace: true },
      );
    });
  }, [setSearchParams, props, paramObservable]);

  return {
    params,
    updateParams,
  };
};
