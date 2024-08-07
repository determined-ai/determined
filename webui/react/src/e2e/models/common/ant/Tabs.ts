import { BaseComponent, NamedComponent } from 'playwright-page-model-base/BaseComponent';

/**
 * Represents the Tabs component from antd/es/tabs/index.js
 */
export class Tabs extends NamedComponent {
  readonly defaultSelector = 'div.ant-tabs';
  readonly tablist = new BaseComponent({ parent: this, selector: '.ant-tabs-nav' });
  readonly tabContent = new BaseComponent({
    parent: this,
    selector: '.ant-tabs-content-holder',
  });

  /**
   * Returns a Tab in the Tabs component
   * @param {string} id - the id of the menu item
   */
  tab(id: string): BaseComponent {
    return new BaseComponent({
      parent: this,
      selector: `div.ant-tabs-tab-btn[id$="${id}"]`,
    });
  }
}
