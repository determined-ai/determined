/*
 * jest-dom adds custom jest matchers for asserting on DOM nodes.
 * allows you to do things like:
 * expect(element).toHaveTextContent(/react/i)
 * learn more: https://github.com/testing-library/jest-dom
 */
import '@testing-library/jest-dom/extend-expect';
import 'micro-observables/batchingForReactDom';
import 'shared/prototypes';
import 'whatwg-fetch';

import Schema from 'async-validator';

import { noOp } from 'shared/utils/service';

/**
 * To clean up the async-validator console warning that get generated during testing.
 * https://github.com/yiminghe/async-validator#how-to-avoid-global-warning
 */
Schema.warning = noOp;

Object.defineProperty(window, 'matchMedia', {
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

global.ResizeObserver = require('resize-observer-polyfill');
