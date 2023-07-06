import { isRight } from 'fp-ts/Either';
import { Map } from 'immutable';
import { Type } from 'io-ts';
import { Observable, WritableObservable } from 'micro-observables';

import { getUserSetting, updateUserSetting } from 'services/api';
import { V1GetUserSettingResponse } from 'services/api-ts-sdk';
import handleError, { ErrorType } from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { observable } from 'utils/observable';

import PollingStore from './polling';

type UserSettings = Record<string, unknown>;
type UserSettingsMap = Map<string, UserSettings>;

const DEFAULT_PATH = 'global';

class UserSettingsStore extends PollingStore {
  #settings: WritableObservable<Loadable<UserSettingsMap>> = observable(NotLoaded);

  public get<T>(path: string | undefined, key: string, type: Type<T>): Observable<T | undefined> {
    return this.#settings.select((loadable) => {
      return Loadable.quickMatch(loadable, undefined, (map) => {
        const pathKey = path || DEFAULT_PATH;
        const pathSettings = map.get(pathKey) || {};
        const value = pathSettings[key] as T | undefined;
        return UserSettingsStore.validateValue<T>(type, value);
      });
    });
  }

  public set<T>(path: string | undefined, key: string, type: Type<T>, value: T) {
    this.#settings.update((loadable) => {
      return Loadable.map(loadable, (map) => {
        const pathKey = path || DEFAULT_PATH;
        const pathSettings = map.get(pathKey) || {};
        const validatedValue = UserSettingsStore.validateValue<T>(type, value);
        const newValue = validatedValue === value ? validatedValue : undefined;
        this.updateUserSetting<T>(pathKey, key, newValue);
        return map.set(pathKey, { ...pathSettings, [key]: newValue });
      });
    });
  }

  public setBatch(
    path: string | undefined,
    batches: Array<{
      key: string;
      type: Type<unknown>;
      value: unknown;
    }> = [],
  ) {
    this.#settings.update((loadable) => {
      return Loadable.map(loadable, (map) => {
        const pathKey = path || DEFAULT_PATH;
        const pathSettings = map.get(pathKey) || {};
        const newSettings: UserSettings = {};
        for (const batch of batches) {
          const validatedValue = UserSettingsStore.validateValue(batch.type, batch.value);
          const newValue = validatedValue === batch.value ? validatedValue : undefined;
          this.updateUserSetting(pathKey, batch.key, newValue);
          newSettings[batch.key] = newValue;
        }
        return map.set(pathKey, { ...pathSettings, ...newSettings });
      });
    });
  }

  public remove(path: string | undefined, key: string) {
    this.removeBatch(path, [key]);
  }

  public removeBatch(path: string | undefined, keys: string[]) {
    this.#settings.update((loadable) => {
      return Loadable.map(loadable, (map) => {
        return map.removeAll(keys);
      });
    });
  }

  public reset() {
    this.#settings.set(NotLoaded);
  }

  protected async poll() {
    const response = await getUserSetting({ signal: this.canceler?.signal });
    this.updateSettingsFromResponse(response);
  }

  protected updateSettingsFromResponse(response: V1GetUserSettingResponse) {
    this.#settings.update((loadable) => {
      const newSettings: UserSettingsMap = Loadable.getOrElse(Map(), loadable);

      for (const setting of response.settings) {
        const pathKey = (setting.storagePath || DEFAULT_PATH).replace(/u:2\//g, '');
        const oldPathSettings = newSettings.get(pathKey) || {};
        const newPathSettings = {
          [setting.key]: setting.value ? JSON.parse(setting.value) : undefined,
        };
        newSettings.set(pathKey, { ...oldPathSettings, ...newPathSettings });
      }

      return Loaded(newSettings);
    });
  }

  protected updateUserSetting<T>(path: string, key: string, value?: T) {
    updateUserSetting({
      setting: { key, storagePath: path, value: JSON.stringify(value) },
      storagePath: path,
    }).catch((e) =>
      handleError(e, {
        isUserTriggered: false,
        publicMessage: `Unable to update user settings for path: ${path}, key: ${key}.`,
        publicSubject: 'Some POST user settings failed.',
        silent: true,
        type: ErrorType.Api,
      }),
    );
  }

  protected static validateValue<T>(type: Type<T>, value?: T): T | undefined {
    return isRight(type.decode(value)) ? value : undefined;
  }
}

export default new UserSettingsStore();
