import React, { ReactElement, ReactNode } from 'react';

import { StoreProvider as UIProvider } from 'stores/contexts/UI';

export const StoreProvider = ({ children }: { children: ReactNode }): ReactElement => (
  <UIProvider>{children}</UIProvider>
);
