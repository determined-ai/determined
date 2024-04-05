import { BaseComponent } from 'e2e/models/BaseComponent';
import { BasePage } from 'e2e/models/BasePage';
import { Pivot } from 'e2e/models/hew/Pivot';

/**
 * Returns a representation of an Admin page.
 * This constructor represents the contents in src/pages/Admin/UserManagement.tsx.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export abstract class AdminPage extends BasePage {
  readonly pivot: Pivot = new Pivot({ parent: this });
  readonly userTab: BaseComponent = new BaseComponent({
    parent: this,
    selector: Pivot.selectorTemplateTabs('user-management'),
  });
  readonly groupTab: BaseComponent = new BaseComponent({
    parent: this,
    selector: Pivot.selectorTemplateTabs('group-management'),
  });
  readonly content: BaseComponent = new BaseComponent({
    parent: this,
    selector: '.ant-tabs-content-holder',
  });
}
