import { Toast } from 'e2e/models/hew/Toast';

/**
 * Returns a representation of the error util.
 * This constructor represents the contents in src/utils/error.ts.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this error
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export const ErrorComponent = Toast;
export type ErrorComponent = Toast;
