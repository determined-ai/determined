import { BaseComponent } from 'playwright-page-model-base/BaseComponent';

import { Card } from 'e2e/models/common/hew/Card';

import { ProjectActionDropdown } from './ProjectActionDropdown';

/**
 * Represents the ProjectCard in the WorkspaceProjects component
 */
export class ProjectCard extends Card {
  override readonly actionMenu = new ProjectActionDropdown({
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
