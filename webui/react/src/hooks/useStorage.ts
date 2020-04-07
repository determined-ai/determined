import { useState } from 'react';

import { Storage, Store } from 'utils/storage';

export const useStorage = (basePath: string, store: Store = window.localStorage): Storage => {
  const [ storage ] = useState(
    new Storage({ basePath, store }),
  );
  return storage;
};

export default useStorage;
