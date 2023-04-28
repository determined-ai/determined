import { Map } from 'immutable';
import { observable } from 'micro-observables';
import { createRoot } from 'react-dom/client';
import { HelmetProvider } from 'react-helmet-async';
import { BrowserRouter } from 'react-router-dom';
import 'uplot/dist/uPlot.min.css';

import css from '../src/App.module.scss';
import { ConfirmationProvider } from '../src/components/kit/useConfirm';
import ThemeProvider from '../src/components/ThemeProvider';
import { Settings, UserSettings } from '../src/hooks/useSettingsProvider';
import DesignKit from '../src/pages/DesignKit';
import { StoreProvider as UIProvider } from '../src/shared/contexts/stores/UI';

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
            <ConfirmationProvider>
              <div className={css.base}>
                <DesignKit />
              </div>
            </ConfirmationProvider>
          </ThemeProvider>
        </UserSettings.Provider>
      </UIProvider>
    </HelmetProvider>
  </BrowserRouter>,
);
