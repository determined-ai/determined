import { useState } from 'react';

import Auth from 'contexts/Auth';
import { Storage, Store } from 'utils/storage';

export const useStorage = (basePath: string, store: Store = window.localStorage): Storage => {
  const auth = Auth.useStateContext();
  const userNamespace = auth.user ? `u:${auth.user.username}` : '';
  const [ storage ] = useState(
    new Storage({ basePath: `${userNamespace}/${basePath}`, store }),
  );
  return storage;
};

export default useStorage;
