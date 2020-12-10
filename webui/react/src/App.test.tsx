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

/*
 * Mocking window.matchMedia used by FormItem from Ant Design
 */
Object.defineProperty(window, 'matchMedia', {
  value: jest.fn().mockImplementation(query => ({
    addEventListener: jest.fn(),
    addListener: jest.fn(), // deprecated
    dispatchEvent: jest.fn(),
    matches: false,
    media: query,
    onchange: null,
    removeEventListener: jest.fn(),
    removeListener: jest.fn(), // deprecated
  })),
  writable: true,
});

/*
 * Mocking useResize hook.
 */
jest.mock('hooks/useResize', () => {
  return jest.fn(() => ({
    height: 1024,
    width: 768,
    x: 0,
    y: 0,
  }));
});

describe('App', () => {
  it('renders app', () => {
    const { container } = render(<MemoryRouter><App /></MemoryRouter>);
    expect(container).toBeInTheDocument();
  });
});
