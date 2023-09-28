import { isRight, match } from 'fp-ts/Either';
import { pipe } from 'fp-ts/function';
import { Map } from 'immutable';
import * as t from 'io-ts';

import { Loadable, Loaded, NotLoaded } from 'components/kit/utils/loadable';
import { getUserSetting, resetUserSetting, updateUserSetting } from 'services/api';
import { V1GetUserSettingResponse, V1UserWebSetting } from 'services/api-ts-sdk';
import { Json, JsonObject } from 'types';
import { isJsonObject, isObject } from 'utils/data';
import handleError, { DetError, ErrorType } from 'utils/error';
import { observable, Observable, WritableObservable } from 'utils/observable';

import PollingStore from './polling';

type State = Map<string, Json>;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function isTypeC(codec: t.Encoder<any, any>): codec is t.TypeC<t.Props> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (codec as any)._tag === 'InterfaceType';
}

/**
 * This stores per-user settings. These are values that affect how the UI functions
 * and are limited in scope to the logged-in user.
 */
export class UserSettingsStore extends PollingStore {
  readonly #settings: WritableObservable<Loadable<State>> = observable(NotLoaded);
  #updates: Promise<void | DetError>[] = [];

  /**
   *
   * @param type Type of the value to be returned or a decoder from JSON to that type
   * @param key Unique key to store and retrieve the settings
   * @returns An observable of the setting value. If that setting has never been set
   *          (or has been removed) this is `null`.
   */
  public get<T>(type: t.Type<T>, key: string): Observable<Loadable<T | null>>;
  public get<T>(type: t.Decoder<Json, T>, key: string): Observable<Loadable<T | null>>;
  public get<T>(type: t.Decoder<Json, T>, key: string): Observable<Loadable<T | null>> {
    return this.#settings.select((settings) => {
      return Loadable.map(settings, (settings) => {
        const value = settings.get(key);
        if (value === undefined) {
          return null;
        }
        return pipe(
          value,
          type.decode,
          match(
            () => {
              console.error(`Setting at key '${key}' could not be decoded as ${type.name}`);
              return null; // Silently swallow decoding errors
            },
            (v) => v,
          ),
        );
      });
    });
  }

  public getAll(): Observable<Loadable<State>> {
    return this.#settings.readOnly();
  }

  public async overwrite(settings: State): Promise<void> {
    const settingsArray = Object.entries(settings.toJS()).flatMap(([storagePath, settings]) =>
      !!settings && isObject(settings)
        ? Object.entries(settings).map(([key, value]) => ({
            key,
            storagePath,
            value: JSON.stringify(value),
          }))
        : [],
    );

    try {
      await resetUserSetting({});
      await updateUserSetting({ settings: settingsArray });
      this.#settings.set(Loaded(settings));
    } catch (error) {
      handleError(error, {
        isUserTriggered: false,
        publicMessage: 'Unable to update user settings, try again later.',
        type: ErrorType.Api,
      });
    }
  }

  public async clear(): Promise<void> {
    try {
      await resetUserSetting({});
      this.#settings.set(Loaded(Map()));
    } catch (error) {
      handleError(error, {
        isUserTriggered: false,
        publicMessage: 'Unable to reset user settings, try again later.',
        type: ErrorType.Api,
      });
    }
  }

  /**
   * This sets the value of a setting and persists it for future sessions.
   * @param type The type of the value or an encoder of the value to JSON.
   * @param key Unique key to store and retrieve the settings
   * @param value New value of the setting
   */
  public set<T>(type: t.Type<T>, key: string, value: T): void;
  public set<T>(type: t.Encoder<T, Json>, key: string, value: T): void;
  public set<T>(type: t.Encoder<T, Json>, key: string, value: T): void {
    const encodedValue: Json = type.encode(value);
    // This is non-blocking. If the API call fails we don't want to block
    // the user from interacting, just let them know that their settings
    // are not persisting. It's also important to update the value immediately
    // for good rendering performance.
    this.updateUserSetting(key, encodedValue);
    this.#settings.update((settings) => {
      return Loadable.map(settings, (settings) => {
        return settings.set(key, encodedValue);
      });
    });
  }

  /**
   * Like set but allows you to pass a partial value and
   * if will be merged with the previous value.
   * @param type The type of the value or an encoder of the value to JSON.
   * @param key Unique key to store and retrieve the settings
   * @param value New value of the setting
   */
  public setPartial<T extends t.Props>(
    type: t.TypeC<T>,
    key: string,
    value: t.TypeOfPartialProps<T>,
  ): void;
  public setPartial<T extends t.Props, U extends t.Props>(
    type: t.IntersectionType<[t.TypeC<T>, t.PartialC<U>]>,
    key: string,
    value: t.TypeOfPartialProps<T> & t.TypeOfPartialProps<U>,
  ): void;
  public setPartial<T>(type: t.Type<T>, key: string, value: T): void {
    const encodedValue = (() => {
      if (isTypeC(type)) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        return t.partial(type.props).encode(value as any);
      }
      // by exclusion the type must be this specific intersection
      // if any new overloads are added this next line is no longer valid
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const i = type as t.IntersectionType<[t.TypeC<any>, t.PartialC<any>]>;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return t.intersection([t.partial(i.types[0].props), i.types[1]]).encode(value as any);
    })();
    // This is non-blocking. If the API call fails we don't want to block
    // the user from interacting, just let them know that their settings
    // are not persisting. It's also important to update the value immediately
    // for good rendering performance.
    this.updateUserSetting(key, encodedValue);
    this.#settings.update((settings) => {
      return Loadable.map(settings, (settings) => {
        return settings.update(key, (oldValue) => {
          const old: JsonObject =
            oldValue && isJsonObject(oldValue) ? oldValue : ({} as JsonObject);
          return { ...old, ...encodedValue };
        });
      });
    });
  }

  /**
   * This updates the value of a setting and persists it for future sessions.
   * @param type The type of the value or an encoder of the value to JSON.
   * @param key Unique key to store and retrieve the settings
   * @param fn Function to update the value of the setting at `key`
   */
  public update<T>(type: t.Type<T, Json>, key: string, fn: (value: T | undefined) => T): void {
    this.#settings.update((settings) => {
      return Loadable.map(settings, (settings) => {
        return settings.update(key, (jsonValue) => {
          let value: T | undefined = undefined;
          if (jsonValue !== undefined) {
            const attempt = type.decode(jsonValue);
            // Silently discard incorrectly formatted values
            if (isRight(attempt)) {
              value = attempt.right;
            } else {
              console.error(`Setting at key '${key}' could not be decoded as ${type.name}`);
            }
          }
          const newValue = fn(value);
          // This is non-blocking. If the API call fails we don't want to block
          // the user from interacting, just let them know that their settings
          // are not persisting. It's also important to update the value immediately
          // for good rendering performance.
          this.updateUserSetting(key, newValue);
          return type.encode(newValue);
        });
      });
    });
  }

  /** Clears the setting and removes the entry entirely from the database. */
  public remove(key: string): void {
    this.#settings.update((loadable) => {
      return Loadable.map(loadable, (map) => {
        /**
         * Construct a dictionary to specifically target specific settings keys to clear out.
         * To clear out an object setting, an object of `{ [key]: undefined }` must be used instead of
         * `null` to clear out each keyed setting in the object.
         */
        const value = map.get(key);
        const resetValue = isObject(value)
          ? Object.keys(value as object).reduce((acc, settingsKey) => {
              acc[settingsKey] = undefined;
              return acc;
            }, {} as Record<string, undefined>)
          : null;
        this.updateUserSetting(key, resetValue);

        return map.removeAll(key);
      });
    });
  }

  /**
   * This resets the store to its initial state, useful for logging the user out.
   */
  public reset(): void {
    this.#settings.set(NotLoaded);
  }

  protected async poll(): Promise<void> {
    try {
      // Wait 500ms for any in-flight updates to finish before getting new state
      await Promise.race([
        Promise.allSettled(this.#updates),
        new Promise((resolve) => setTimeout(resolve, 500)),
      ]);
      const response = await getUserSetting({ signal: this.canceler?.signal });
      this.updateSettingsFromResponse(response);
    } catch (error) {
      handleError(error, {
        isUserTriggered: false,
        publicMessage: 'Unable to fetch user settings, try refreshing.',
        type: ErrorType.Api,
      });
    } finally {
      this.#settings.update((settings) =>
        Loadable.match(settings, {
          Loaded: (settings) => Loaded(settings),
          // If we are unable to load settings just notify the user and unblock them.
          NotLoaded: () => Loaded(Map()),
        }),
      );
    }
  }

  protected updateSettingsFromResponse(response: V1GetUserSettingResponse): void {
    this.#settings.update((loadable) => {
      let newSettings: State = Loadable.getOrElse(Map(), loadable);

      newSettings = newSettings.withMutations((newSettings) => {
        for (const setting of response.settings) {
          const pathKey = setting.storagePath || setting.key;
          const oldPathSettings = newSettings.get(pathKey);
          if (oldPathSettings && isJsonObject(oldPathSettings)) {
            const newPathSettings = {
              [setting.key]: setting.value ? JSON.parse(setting.value) : undefined,
            };
            newSettings.set(pathKey, { ...oldPathSettings, ...newPathSettings });
          } else if (setting.key === '_ROOT') {
            newSettings.set(pathKey, setting.value ? JSON.parse(setting.value) : undefined);
          } else {
            newSettings.set(pathKey, {
              [setting.key]: setting.value ? JSON.parse(setting.value) : undefined,
            });
          }
        }
      });
      return Loaded(newSettings);
    });
  }

  // TODOs:
  // - We shouldn't be JSON.stringifying stuff before sending it to the API
  //   DB is perfectly capable of storing JSON and we'll get better performance
  //   and flexibility that way
  // - API should support setting non-objects as values directly like a regular
  //   key/value store.
  protected updateUserSetting<T>(key: string, value: T): Promise<void | DetError> {
    const dbUpdates: Array<V1UserWebSetting> = [];
    if (isObject(value)) {
      const settings = value as unknown as { [key: string]: unknown };
      dbUpdates.push(
        ...Object.keys(settings).reduce<V1UserWebSetting[]>((acc, setting) => {
          return [
            ...acc,
            {
              key: setting,
              storagePath: key,
              value: JSON.stringify(settings[setting]),
            },
          ];
        }, []),
      );
    } else {
      dbUpdates.push({ key: '_ROOT', storagePath: key, value: JSON.stringify(value) });
    }
    const promise = updateUserSetting({ settings: dbUpdates })
      .finally(() => {
        this.#updates = this.#updates.filter((p) => p !== promise);
      })
      .catch((e) =>
        handleError(e, {
          isUserTriggered: false,
          publicMessage: `Unable to update user settings for key: ${key}.`,
          publicSubject: 'Some POST user settings failed.',
          silent: true,
          type: ErrorType.Api,
        }),
      );
    this.#updates = [...this.#updates, promise];
    return promise;
  }

  /**
   * DO NOT USE
   *
   * This is a temporary bridge for the old useSettings function
   */
  public _forUseSettingsOnly(): WritableObservable<Loadable<State>> {
    return this.#settings;
  }
}

export default new UserSettingsStore();
