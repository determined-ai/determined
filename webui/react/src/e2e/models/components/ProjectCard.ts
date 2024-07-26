import { BaseComponent } from 'e2e/models/common/base/BaseComponent';
import { Card } from 'e2e/models/common/hew/Card';

import { ProjectActionDropdown } from './ProjectActionDropdown';

/**
 * Represents the ProjectsCard in the WorkspaceProjects component
 */
export class ProjectsCard extends Card {
  override readonly actionMenu = new ProjectActionDropdown({
    clickThisComponentToOpen: this.actionMenuContainer,
    root: this.root,
  });
  readonly archivedBadge = new BaseComponent({
    parent: this,
    selector: '[data-testid="archived"]',
  });
}
