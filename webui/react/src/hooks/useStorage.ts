import { useState } from 'react';

import { resetUserSetting } from 'services/api';
import userStore from 'stores/users';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';
import { StorageManager } from 'utils/storage';

export const userPreferencesStorage = (): (() => void) => {
  const storage = new StorageManager({ basePath: 'u', delimiter: ':', store: window.localStorage });
  const resetStorage = async () => {
    await resetUserSetting({});
    storage.reset();
  };
  return resetStorage;
};

export const useStorage = (
  basePath: string,
  store: Storage = window.localStorage,
): StorageManager => {
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const userNamespace = currentUser ? `u:${currentUser.id}` : '';
  const [storage] = useState(
    new StorageManager({ basePath: `${userNamespace}/${basePath}`, store }),
  );
  return storage;
};

export default useStorage;
