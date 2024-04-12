import { BaseComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Switch item component from Ant.
 * .
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Dropdown
 * @param {string} obj.selector - the selector for the entire switch. The finding the button or label is handled by this component.
 */
export class Switch extends BaseComponent {
  readonly switch: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'button[role="switch"]',
  });
  readonly label: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'label',
  });
}
