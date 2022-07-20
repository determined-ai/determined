import React from 'react';
import { createRoot } from 'react-dom/client';
import { Router } from 'react-router-dom';

import history from 'shared/routes/history';

/* Import the styles first to allow components to override styles. */
import 'shared/styles/index.scss';
import 'uplot/dist/uPlot.min.css';

import App from './App';
import * as serviceWorker from './serviceWorker';
import 'shared/prototypes';
import 'dev';

const container = document.getElementById('root');
// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
const root = createRoot(container!);

root.render(
  // <React.StrictMode>
  <Router history={history}>
    <App />
  </Router>,
  // </React.StrictMode>,
);

/*
 * If you want your app to work offline and load faster, you can change
 * unregister() to register() below. Note this comes with some pitfalls.
 * Learn more about service workers: https://bit.ly/CRA-PWA
 */
serviceWorker.unregister();
