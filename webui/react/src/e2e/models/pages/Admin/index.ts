import { BasePage } from 'e2e/models/common/base/BasePage';
import { Pivot } from 'e2e/models/common/hew/Pivot';

/**
 * Represents the admin page from src/pages/Admin/index.tsx
 */
export abstract class AdminPage extends BasePage {
  readonly title = '';
  readonly pivot = new Pivot({ parent: this });
  readonly userTab = this.pivot.tab('user-management');
  readonly groupTab = this.pivot.tab('group-management');
}
