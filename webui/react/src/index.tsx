import React from 'react';
import ReactDOM from 'react-dom';
import { Router } from 'react-router-dom';
import { CompatRouter } from 'react-router-dom-v5-compat';

import history from 'shared/routes/history';

/* Import the styles first to allow components to override styles. */
import 'shared/styles/index.scss';
import 'uplot/dist/uPlot.min.css';

import App from './App';
import * as serviceWorker from './serviceWorker';
import 'shared/prototypes';
import 'dev';

ReactDOM.render(
  <React.StrictMode>
    <Router history={history}>
      <CompatRouter>
        <App />
      </CompatRouter>
    </Router>
  </React.StrictMode>,
  document.getElementById('root'),
);

/*
 * If you want your app to work offline and load faster, you can change
 * unregister() to register() below. Note this comes with some pitfalls.
 * Learn more about service workers: https://bit.ly/CRA-PWA
 */
serviceWorker.unregister();
