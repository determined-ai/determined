import 'micro-observables/batchingForReactDom';
import React from 'react';
import { createRoot } from 'react-dom/client';
/**
 * It's considered unstable until `react-router-dom` can detect
 * history version mismatches when supplying your own history.
 * https://reactrouter.com/en/v6.3.0/api#unstable_historyrouter
 */
import { createBrowserRouter, RouterProvider } from 'react-router-dom';

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

const container = document.getElementById('root');
// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
const root = createRoot(container!);

const router = createBrowserRouter(
  [
    // match everything with "*"
    { element: <App />, path: '*' },
  ],
  { basename: process.env.PUBLIC_URL },
);

root.render(
  // <React.StrictMode>
  <RouterProvider router={router} />,
  // </React.StrictMode>,
);

/*
 * If you want your app to work offline and load faster, you can change
 * unregister() to register() below. Note this comes with some pitfalls.
 * Learn more about service workers: https://bit.ly/CRA-PWA
 */
serviceWorker.unregister();
