import { UIProvider } from 'hew/Theme';
import React, { ReactElement, ReactNode } from 'react';

export const StoreProvider = ({ children }: { children: ReactNode }): ReactElement => (
  <UIProvider>{children}</UIProvider>
);
