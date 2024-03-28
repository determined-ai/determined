import { BaseComponent, NamedComponent, NamedComponentArgs } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Mondal component from Hew.
 * This constructor represents the contents in hew/src/kit/Toast.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Toast
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */

export class Toast extends NamedComponent {
  static defaultSelector = '.ant-notification';
  static readonly selectorTopRight = '.ant-notification-topRight';
  static readonly selectorBottomRight = '.ant-notification-bottomRight';
  constructor({ parent, selector }: NamedComponentArgs) {
    super({ parent: parent, selector: selector || Toast.defaultSelector });
  }
  readonly alert: BaseComponent = new BaseComponent({ parent: this, selector: "[role='alert']" });
  readonly message: BaseComponent = new BaseComponent({
    parent: this.alert,
    selector: '.ant-notification-notice-message',
  });
  readonly description: BaseComponent = new BaseComponent({
    parent: this.alert,
    selector: '.ant-notification-notice-description',
  });
}
