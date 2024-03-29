import { BaseComponent, NamedComponent, NamedComponentArgs } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Mondal component from Hew.
 * This constructor represents the contents in hew/src/kit/Pivot.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Pivot
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class Pivot extends NamedComponent {
  static defaultSelector = 'div.ant-tabs';
  constructor({ selector, parent }: NamedComponentArgs) {
    super({ parent: parent, selector: selector || Pivot.defaultSelector });
  }
  static selectorTemplateTabs(id: string): string {
    return `div.ant-tabs-tab-btn[id$="${id}"]`;
  }
  readonly tablist: BaseComponent = new BaseComponent({ parent: this, selector: '.ant-tabs-nav' });
  readonly tabContent: BaseComponent = new BaseComponent({
    parent: this,
    selector: '.ant-tabs-content-holder',
  });
}
