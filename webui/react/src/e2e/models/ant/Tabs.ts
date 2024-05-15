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
  readonly tablist = new BaseComponent({ parent: this, selector: '.ant-tabs-nav' });
  readonly tabContent = new BaseComponent({
    parent: this,
    selector: '.ant-tabs-content-holder',
  });

  /**
   * Returns a representation of a tab item with the specified id.
   * @param {string} id - the id of the menu item
   */
  tab(id: string): BaseComponent {
    return new BaseComponent({
      parent: this.tablist,
      selector: `div.ant-tabs-tab-btn[id$="${id}"]`,
    });
  }
}
