import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Mondal component from Hew.
 * This constructor represents the contents in hew/src/kit/Pivot.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this Pivot
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class Pivot extends NamedComponent {
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
    selector: '.ant-tabs-content-holder role="tabpanel".ant-tabs-tabpane-active',
  });
}
