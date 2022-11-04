/*
 * jest-dom adds custom jest matchers for asserting on DOM nodes.
 * allows you to do things like:
 * expect(element).toHaveTextContent(/react/i)
 * learn more: https://github.com/testing-library/jest-dom
 */
import '@testing-library/jest-dom/extend-expect';
import 'shared/prototypes';
import { readFileSync } from 'fs';

import Schema from 'async-validator';

import { noOp } from 'shared/utils/service';

/**
 * To clean up the async-validator console warning that get generated during testing.
 * https://github.com/yiminghe/async-validator#how-to-avoid-global-warning
 */
Schema.warning = noOp;

Object.defineProperty(window, 'matchMedia', {
  value: () => ({
    addEventListener: jest.fn(),
    addListener: jest.fn(), // deprecated
    dispatchEvent: jest.fn(),
    matches: false,
    onchange: null,
    removeEventListener: jest.fn(),
    removeListener: jest.fn(), // deprecated
  }),
});

Object.defineProperty(window, 'loadAntdStyleSheet', {
  /**
   * function to load ant styles into test environment
   * https://github.com/testing-library/jest-dom/issues/113#issuecomment-496971128
   * https://github.com/testing-library/jest-dom
   * /blob/09f7f041805b2a4bcf5ac5c1e8201ee10a69ab9b/src/__tests__/to-have-style.js#L12-L18
   */
  value: () => {
    const antdStyleSheet = readFileSync('node_modules/antd/dist/antd.css').toString();
    const style = document.createElement('style');
    style.innerHTML = antdStyleSheet;
    document.body.appendChild(style);
  },
});

global.ResizeObserver = require('resize-observer-polyfill');
