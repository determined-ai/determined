import { useState } from 'react';

import { useStore } from 'contexts/Store';
import { resetUserSetting } from 'services/api';
import { Storage, Store } from 'shared/utils/storage';

export const userPreferencesStorage = (): () => void => {

  const storage = new Storage(
    { basePath: 'u', delimiter: ':', store: window.localStorage },
  );
  const resetStorage = async () => {
    await resetUserSetting({});
    storage.reset();
  };
  return resetStorage;
};

export const useStorage = (basePath: string, store: Store = window.localStorage): Storage => {
  const { auth } = useStore();
  const userNamespace = auth.user ? `u:${auth.user.id}` : '';
  const [ storage ] = useState(
    new Storage({ basePath: `${userNamespace}/${basePath}`, store }),
  );
  return storage;
};

export default useStorage;
