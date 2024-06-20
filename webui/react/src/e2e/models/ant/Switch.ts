import { BaseComponent } from 'e2e/models/base/BaseComponent';

/**
 * Represents the Switch component from antd/es/switch/index.js
 */
export class Switch extends BaseComponent {
  readonly switch = new BaseComponent({
    parent: this,
    selector: 'button[role="switch"]',
  });
  readonly label = new BaseComponent({
    parent: this,
    selector: 'label',
  });
}
