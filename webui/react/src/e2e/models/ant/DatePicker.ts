import { NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the DatePicker component from Hew.
 * This constructor represents the contents in hew/src/kit/DatePicker.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this DatePicker
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class DatePicker extends NamedComponent {
  readonly defaultSelector = '[class^="DatePicker_base"]';
  // TODO implement this
}
