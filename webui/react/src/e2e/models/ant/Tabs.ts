import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Tabs component from Ant.
 * This constructor represents the contents in antd/es/tabs/index.d.ts.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this Tabs
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class Tabs extends NamedComponent {
  readonly defaultSelector = 'div.ant-tabs';

  /**
   * Returns a templated selector for children components.
   * @param {string} id - tab id
   */
  static selectorTemplateTabs(id: string): string {
    return `div.ant-tabs-tab-btn[id$="${id}"]`;
  }
  readonly tablist = new BaseComponent({ parent: this, selector: '.ant-tabs-nav' });
  readonly tabContent = new BaseComponent({
    parent: this,
    selector: '.ant-tabs-content-holder',
  });
}
