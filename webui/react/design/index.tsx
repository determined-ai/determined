import React from 'react';
import { Map } from 'immutable';
import { observable } from 'micro-observables';
import { createRoot } from 'react-dom/client';
import { HelmetProvider } from 'react-helmet-async';
import { createBrowserRouter, RouterProvider } from 'react-router-dom';
import 'uplot/dist/uPlot.min.css';

import css from '../src/App.module.scss';
import { ConfirmationProvider } from '../src/components/kit/useConfirm';
import ThemeProvider from '../src/components/ThemeProvider';
import { Settings, UserSettings } from '../src/hooks/useSettingsProvider';
import DesignKit from '../src/pages/DesignKit';
import { StoreProvider as UIProvider } from '../src/stores/contexts/UI';
import { Loaded } from '../src/utils/loadable';

import 'antd/dist/reset.css';

const fakeSettingsContext = {
  clearQuerySettings: () => undefined,
  isLoading: false,
  querySettings: '',
  state: observable(Loaded(Map<string, Settings>())),
};

const router = createBrowserRouter([
  {
    path: "*",
    element: <HelmetProvider>
      <UIProvider>
        <UserSettings.Provider value={fakeSettingsContext}>
          <ThemeProvider>
            <ConfirmationProvider>
              <div className={css.base}>
                <DesignKit />
              </div>
            </ConfirmationProvider>
          </ThemeProvider>
        </UserSettings.Provider>
      </UIProvider>
    </HelmetProvider>,
  },
]);

// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
createRoot(document.getElementById('root')!).render(
  <RouterProvider router={router} />
);
