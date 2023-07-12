import { match } from 'fp-ts/Either';
import { pipe } from 'fp-ts/function';
import { Map } from 'immutable';
import * as t from 'io-ts';

import { getUserSetting, updateUserSetting } from 'services/api';
import { V1GetUserSettingResponse } from 'services/api-ts-sdk';
import { UpdateUserSettingParams } from 'services/types';
import { Json, JsonObject } from 'types';
import { isJsonObject, isObject } from 'utils/data';
import handleError, { ErrorType } from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
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
class UserSettingsStore extends PollingStore {
  readonly #settings: WritableObservable<Loadable<State>> = observable(NotLoaded);

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
            () => null, // Silently swallow decoding errors
            (v) => v,
          ),
        );
      });
    });
  }

  /**
   * This sets the value of a setting and persists it for future sessions.
   * If the setting value is an object you can pass a partial value and
   * if will be merged with the previous value.
   * @param type The type of the value or an encoder of the value to JSON.
   * @param key Unique key to store and retrieve the settings
   * @param value New value of the setting
   */
  public set<T>(type: t.Type<T>, key: string, value: T): void;
  public set<T extends t.Props>(type: t.TypeC<T>, key: string, value: Partial<T>): void;
  public set<T>(type: t.Encoder<T, Json>, key: string, value: T): void;
  public set<T>(type: t.Encoder<T, Json> | t.TypeC<t.Props>, key: string, value: T): void {
    if (isTypeC(type)) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const encodedValue = t.partial(type.props).encode(value as any);
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
    } else {
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
  }

  /** Clears the setting, returning it to `null`. */
  public remove(key: string) {
    this.updateUserSetting(key, null);
    this.#settings.update((loadable) => {
      return Loadable.map(loadable, (map) => {
        return map.removeAll(key);
      });
    });
  }

  /**
   * This resets the store to its initial state, useful for logging the user out.
   */
  public reset() {
    this.#settings.set(NotLoaded);
  }

  protected async poll() {
    try {
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

  protected updateSettingsFromResponse(response: V1GetUserSettingResponse) {
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
  protected updateUserSetting<T>(key: string, value: T) {
    const dbUpdates: Array<UpdateUserSettingParams> = [];
    if (isObject(value)) {
      const settings = value as unknown as { string: unknown };
      dbUpdates.push(
        ...Object.keys(settings).reduce<UpdateUserSettingParams[]>((acc, setting) => {
          return [
            ...acc,
            {
              setting: {
                key: setting,
                storagePath: key,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                value: JSON.stringify((settings as any)[setting]),
              },
              storagePath: key,
            },
          ];
        }, []),
      );
    } else {
      dbUpdates.push({
        setting: { key: '_ROOT', storagePath: key, value: JSON.stringify(value) },
        storagePath: key,
      });
    }
    Promise.allSettled(
      dbUpdates.map((update) => {
        return updateUserSetting(update);
      }),
    ).catch((e) =>
      handleError(e, {
        isUserTriggered: false,
        publicMessage: `Unable to update user settings for key: ${key}.`,
        publicSubject: 'Some POST user settings failed.',
        silent: true,
        type: ErrorType.Api,
      }),
    );
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
