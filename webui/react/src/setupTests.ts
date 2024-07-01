/*
 * jest-dom adds custom jest matchers for asserting on DOM nodes.
 * allows you to do things like:
 * expect(element).toHaveTextContent(/react/i)
 * learn more: https://github.com/testing-library/jest-dom
 */
import '@testing-library/jest-dom/extend-expect';
import 'micro-observables/batchingForReactDom';
import 'whatwg-fetch';

import Schema from 'async-validator';

// this code doesn't work in node environments
if (globalThis.window) {
  await import('utils/prototypes');
  const { noOp } = await import('utils/service');

  /**
   * To clean up the async-validator console warning that get generated during testing.
   * https://github.com/yiminghe/async-validator#how-to-avoid-global-warning
   */
  Schema.warning = noOp;
}

Object.defineProperty(globalThis, 'matchMedia', {
  value: () => ({
    addEventListener: vi.fn(),
    addListener: vi.fn(), // deprecated
    dispatchEvent: vi.fn(),
    matches: false,
    onchange: null,
    removeEventListener: vi.fn(),
    removeListener: vi.fn(), // deprecated
  }),
});

vi.mock('router');
vi.mock('services/api', () => ({}));

global.ResizeObserver = require('resize-observer-polyfill');

// https://vitest.dev/guide/common-errors#cannot-mock-mocked-file-js-because-it-is-already-loaded
vi.resetModules();
