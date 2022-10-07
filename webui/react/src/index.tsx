import React from 'react';
import ReactDOM from 'react-dom';
/**
 * It's considered unstable until `react-router-dom` can detect
 * history version mismatches when supplying your own history.
 * https://reactrouter.com/en/v6.3.0/api#unstable_historyrouter
 */
import { unstable_HistoryRouter as HistoryRouter } from 'react-router-dom';

import history from 'shared/routes/history';

/* Import the styles first to allow components to override styles. */
import 'shared/styles/index.scss';
import 'uplot/dist/uPlot.min.css';

import App from './App';
import * as serviceWorker from './serviceWorker';
import 'shared/prototypes';
import 'dev';

// redirect to basename if needed
if (process.env.PUBLIC_URL && history.location.pathname === '/') {
  history.replace(process.env.PUBLIC_URL);
}

ReactDOM.render(
  <React.StrictMode>
    <HistoryRouter basename={process.env.PUBLIC_URL} history={history}>
      <App />
    </HistoryRouter>
  </React.StrictMode>,
  document.getElementById('root'),
);

/*
 * If you want your app to work offline and load faster, you can change
 * unregister() to register() below. Note this comes with some pitfalls.
 * Learn more about service workers: https://bit.ly/CRA-PWA
 */
serviceWorker.unregister();
