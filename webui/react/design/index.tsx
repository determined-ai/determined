import { Map } from 'immutable';
import { observable } from 'micro-observables';
import { createRoot } from 'react-dom/client';
import { HelmetProvider } from 'react-helmet-async';
import { createBrowserRouter, RouterProvider } from 'react-router-dom';

import 'uplot/dist/uPlot.min.css';
import css from 'App.module.scss';
import { ThemeProvider } from 'components/kit/Theme';
import { ConfirmationProvider } from 'components/kit/useConfirm';
import { Loaded } from 'components/kit/utils/loadable';
import { Settings, UserSettings } from 'hooks/useSettingsProvider';
import DesignKit, { DesignKitContainer } from 'pages/DesignKit';

import 'antd/dist/reset.css';

const fakeSettingsContext = {
  clearQuerySettings: () => undefined,
  isLoading: false,
  querySettings: new URLSearchParams(),
  state: observable(Loaded(Map<string, Settings>())),
};

const router = createBrowserRouter([
  {
    element: (
      <ThemeProvider>
        <DesignKitContainer>
          <HelmetProvider>
            <UserSettings.Provider value={fakeSettingsContext}>
              <ConfirmationProvider>
                <div className={css.base}>
                  <DesignKit />
                </div>
              </ConfirmationProvider>
            </UserSettings.Provider>
          </HelmetProvider>
        </DesignKitContainer>
      </ThemeProvider>
    ),
    path: '*',
  },
]);

// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
createRoot(document.getElementById('root')!).render(<RouterProvider router={router} />);
