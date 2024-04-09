import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Mondal component from Hew.
 * This constructor represents the contents in hew/src/kit/Toast.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this Toast
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */

export class Toast extends NamedComponent {
  readonly defaultSelector = '.ant-notification';
  static readonly selectorTopRight = '.ant-notification-topRight';
  static readonly selectorBottomRight = '.ant-notification-bottomRight';
  readonly alert: BaseComponent = new BaseComponent({ parent: this, selector: '[role="alert"]' });
  readonly message: BaseComponent = new BaseComponent({
    parent: this.alert,
    selector: '.ant-notification-notice-message',
  });
  readonly description: BaseComponent = new BaseComponent({
    parent: this.alert,
    selector: '.ant-notification-notice-description',
  });
}
