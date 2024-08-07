import { BaseComponent, NamedComponent } from 'playwright-page-model-base/BaseComponent';

/**
 * Represents the Notification component from antd/es/notification/index.js
 */
export class Notification extends NamedComponent {
  readonly defaultSelector = '.ant-notification';
  static readonly selectorTopRight = '.ant-notification-topRight';
  static readonly selectorBottomRight = '.ant-notification-bottomRight';
  readonly alert = new BaseComponent({ parent: this, selector: '[role="alert"]' });
  readonly message = new BaseComponent({
    parent: this.alert,
    selector: '.ant-notification-notice-message',
  });
  readonly description = new BaseComponent({
    parent: this.alert,
    selector: '.ant-notification-notice-description',
  });
  readonly close = new BaseComponent({
    parent: this,
    selector: 'a.ant-notification-notice-close',
  });
}
