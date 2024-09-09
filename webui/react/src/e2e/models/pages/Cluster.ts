import { DeterminedPage } from 'e2e/models/common/base/BasePage';
import { DynamicTabs } from 'e2e/models/components/DynamicTabs';
import { PageComponent } from 'e2e/models/components/Page';

/**
 * Represents the Cluster page from src/pages/Cluster.tsx
 */
export class Cluster extends DeterminedPage {
  readonly title = 'Cluster';
  readonly url = 'clusters';
  readonly pageComponent = new PageComponent({ parent: this });
  readonly dynamicTabs = new DynamicTabs({ parent: this.pageComponent });
  readonly overviewTab = this.dynamicTabs.pivot.tab('overview');
  readonly historicalUsageTab = this.dynamicTabs.pivot.tab('historical-usage');
  readonly logsTab = this.dynamicTabs.pivot.tab('logs');
}
