import { BaseComponent } from 'playwright-page-model-base/BaseComponent';

import { Card } from 'e2e/models/common/hew/Card';
import { WorkspaceActionDropdown } from 'e2e/models/components/WorkspaceActionDropdown';

/**
 * Represents the WorkspaceCard src/pages/WorkspaceList/WorkspaceCard.tsx
 */
export class WorkspaceCard extends Card {
  override readonly actionMenu = new WorkspaceActionDropdown({
    clickThisComponentToOpen: this.actionMenuContainer,
    root: this.root,
  });
  readonly archivedBadge = new BaseComponent({
    parent: this,
    selector: '[data-testid="archived"]',
  });
  readonly title = new BaseComponent({
    parent: this,
    selector: 'h1',
  });
}
