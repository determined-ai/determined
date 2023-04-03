import { Map } from 'immutable';
import { observable } from 'micro-observables';
import { createRoot } from 'react-dom/client';
import { HelmetProvider } from 'react-helmet-async';
import { BrowserRouter } from 'react-router-dom';
import 'uplot/dist/uPlot.min.css';

import css from 'App.module.scss';
import ThemeProvider from 'components/ThemeProvider';
import { Settings, UserSettings } from 'hooks/useSettingsProvider';
import DesignKit from 'pages/DesignKit';
import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';

import 'antd/dist/reset.css';

const fakeSettingsContext = {
  clearQuerySettings: () => undefined,
  isLoading: observable(false),
  querySettings: '',
  state: observable(Map<string, Settings>()),
};

// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
createRoot(document.getElementById('root')!).render(
  <BrowserRouter>
    <HelmetProvider>
      <UIProvider>
        <UserSettings.Provider value={fakeSettingsContext}>
          <ThemeProvider>
            <div className={css.base}>
              <DesignKit />
            </div>
          </ThemeProvider>
        </UserSettings.Provider>
      </UIProvider>
    </HelmetProvider>
  </BrowserRouter>,
);
