import React, { ReactElement, ReactNode } from 'react';

import { UIProvider } from 'components/kit/Theme';

export const StoreProvider = ({ children }: { children: ReactNode }): ReactElement => (
  <UIProvider>{children}</UIProvider>
);
