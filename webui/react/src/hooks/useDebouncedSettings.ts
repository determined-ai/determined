import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import * as t from 'io-ts';
import { debounce, isEqual } from 'lodash';
import { useCallback, useLayoutEffect, useMemo, useState } from 'react';

import userSettings from 'stores/userSettings';
import { eagerSubscribe } from 'utils/observable';

/**
 * Helper to handle settings slices that update frequently. In such cases, many
 * settings updates will be sent to the server even though we really only want
 * to persist the last change to the server. Debouncing server requests at the
 * store level is no good for two reasons:
 *
 * - the store assumes that the server state wins, so it introduces a race
 * condition between local state and server state.
 * - all other changes to settings that come in during the debounce period will
     be dropped
 *
 * As such, this hook:
 *
 * - handles the hand-off between local and server state (that is, server state
 *   wins until the initial load, at which point local state wins)
 * - provides local state and an update function which debounces sending
 *   state to the store/server
 */
export function useDebouncedSettings<T extends t.HasProps | t.ExactC<t.HasProps>>(
  type: T,
  path: string,
): [Loadable<t.TypeOf<T> | null>, (p: t.TypeOf<T>) => void] {
  const settingsObs = useMemo(() => userSettings.get(type, path), [type, path]);
  const [localState, updateLocalState] = useState<Loadable<T | null>>(NotLoaded);

  useLayoutEffect(() => {
    return eagerSubscribe(settingsObs, (curSettings, prevSettings) => {
      if (!prevSettings?.isLoaded) {
        curSettings.forEach((s) => {
          updateLocalState(Loaded(s));
        });
      }
    });
  }, [settingsObs]);

  const debouncedPartialUpdate = useMemo(
    () => debounce((p: Partial<T>) => userSettings.setPartial(type, path, p), 200),
    [path, type],
  );

  const updateSettings = useCallback(
    (update: T) => {
      // don't send settings to server if they haven't loaded yet
      settingsObs.get().forEach(() => {
        updateLocalState((localStateLoadable) => {
          return localStateLoadable.flatMap((ls) => {
            const newState = { ...ls, ...update };
            if (isEqual(newState, ls)) {
              return localStateLoadable;
            }
            if (newState) {
              debouncedPartialUpdate(update);
            }
            return Loaded(newState);
          });
        });
      });
    },
    [settingsObs, debouncedPartialUpdate],
  );

  return [localState, updateSettings];
}
