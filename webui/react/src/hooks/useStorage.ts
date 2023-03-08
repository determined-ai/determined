import { useState } from 'react';

import { resetUserSetting } from 'services/api';
import { StorageManager } from 'shared/utils/storage';
import usersStore from 'stores/users';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

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
  const loadableCurrentUser = useObservable(usersStore.getCurrentUser());
  const userNamespace = Loadable.match(loadableCurrentUser, {
    Loaded: (cUser) => (cUser ? `u:${cUser.id}` : ''),
    NotLoaded: () => '',
  });
  const [storage] = useState(
    new StorageManager({ basePath: `${userNamespace}/${basePath}`, store }),
  );
  return storage;
};

export default useStorage;
