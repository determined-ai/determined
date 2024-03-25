import { BaseComponent, NamedComponent, NamedComponentArgs } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the error util.
 * This constructor represents the contents in src/utils/error.ts.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this error
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class ErrorComponent extends NamedComponent {
  static defaultSelector = '.ant-notification';
  static readonly selectorTopRight = '.ant-notification-topRight';
  static readonly selectorBottomRight = '.ant-notification-bottomRight';
  constructor({ parent, selector }: NamedComponentArgs) {
    super({ parent: parent, selector: selector || ErrorComponent.defaultSelector });
  }
  readonly alert: BaseComponent = new BaseComponent({ parent: this, selector: "[role='alert']" });
  readonly message: BaseComponent = new BaseComponent({ parent: this.alert, selector: '.ant-notification-notice-message' });
  readonly description: BaseComponent = new BaseComponent({ parent: this.alert, selector: '.ant-notification-notice-description' });
}
