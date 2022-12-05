import { useState } from 'react';

import { resetUserSetting } from 'services/api';
import { StorageManager } from 'shared/utils/storage';
import { useAuth } from 'stores/users';

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
  const { auth } = useAuth();
  const userNamespace = auth.user ? `u:${auth.user.id}` : '';
  const [storage] = useState(
    new StorageManager({ basePath: `${userNamespace}/${basePath}`, store }),
  );
  return storage;
};

export default useStorage;
