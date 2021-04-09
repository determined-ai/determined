import { useState } from 'react';

import { useStore } from 'contexts/Store';
import { Storage, Store } from 'utils/storage';

export const useStorage = (basePath: string, store: Store = window.localStorage): Storage => {
  const { auth } = useStore();
  const userNamespace = auth.user ? `u:${auth.user.username}` : '';
  const [ storage ] = useState(
    new Storage({ basePath: `${userNamespace}/${basePath}`, store }),
  );
  return storage;
};

export default useStorage;
