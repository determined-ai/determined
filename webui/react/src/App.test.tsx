import { render } from '@testing-library/react';
import React from 'react';
import { MemoryRouter } from 'react-router-dom';
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
    const { container } = render(<MemoryRouter><App /></MemoryRouter>);
    expect(container).toBeInTheDocument();
  });
});
