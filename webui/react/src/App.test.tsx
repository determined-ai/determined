import { render } from '@testing-library/react';
import { createBrowserHistory } from 'history';
import React from 'react';
import { Router } from 'react-router-dom';
import sinon from 'sinon';

import App from './App';

/*
 * The following lines allows the project to take advantage
 * of window.location.assign and window.location.replace
 * without triggering an error from jest and jsdom,
 * especially for CI jest tests.
 */
sinon.stub(window.location, 'assign');
sinon.stub(window.location, 'replace');

describe('App', () => {
  it('renders app', () => {
    const history = createBrowserHistory();
    const { container } = render(<Router history={history}><App /></Router>);
    expect(container).toBeInTheDocument();
  });
});
