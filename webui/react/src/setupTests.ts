/*
 * jest-dom adds custom jest matchers for asserting on DOM nodes.
 * allows you to do things like:
 * expect(element).toHaveTextContent(/react/i)
 * learn more: https://github.com/testing-library/jest-dom
 */
import '@testing-library/jest-dom/extend-expect';
import 'prototypes';

Object.defineProperty(window, 'matchMedia', {
  value: () => ({
    addEventListener: jest.fn(),
    addListener: jest.fn(),         // deprecated
    dispatchEvent: jest.fn(),
    matches: false,
    onchange: null,
    removeEventListener: jest.fn(),
    removeListener: jest.fn(),       // deprecated
  }),
});

global.ResizeObserver = require('resize-observer-polyfill');
