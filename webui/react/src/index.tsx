import { createBrowserHistory } from 'history';
import React from 'react';
import ReactDOM from 'react-dom';
import { Router } from 'react-router-dom';

/* Import the styles first to allow components to override styles. */
import 'styles/index.scss';

import App from './App';
import * as serviceWorker from './serviceWorker';
import 'Dev';

const history = createBrowserHistory();

ReactDOM.render(<Router history={history}><App /></Router>, document.getElementById('root'));

/*
 * If you want your app to work offline and load faster, you can change
 * unregister() to register() below. Note this comes with some pitfalls.
 * Learn more about service workers: https://bit.ly/CRA-PWA
 */
serviceWorker.unregister();
