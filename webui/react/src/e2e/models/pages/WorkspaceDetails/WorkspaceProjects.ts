import { BaseComponent } from 'e2e/models/common/base/BaseComponent';
import { BaseReactFragment } from 'e2e/models/common/base/BaseReactFragment';
import { Card } from 'e2e/models/common/hew/Card';
import { ProjectActionDropdown } from 'e2e/models/components/ProjectActionDropdown';
import { ProjectCreateModal } from 'e2e/models/components/ProjectCreateModal';
import { ProjectDeleteModal } from 'e2e/models/components/ProjectDeleteModal';

/**
 * Represents the WorkspaceProjects page in src/pages/WorkspaceDetails/WorkspaceProjects.tsx
 */
export class WorkspaceProjects extends BaseReactFragment {
  readonly tab = 'projects';
  readonly url = /workspaces\/\d+\/projects/;

  readonly newProject = new BaseComponent({
    parent: this,
    selector: '[data-testid="newProject"]',
  });
  readonly createModal = new ProjectCreateModal({
    root: this.root,
  });
  readonly deleteModal = new ProjectDeleteModal({
    root: this.root,
  });
  cardByName(name: string): ProjectsCard {
    return new ProjectsCard({
      attachment: `[data-testid="card-${name}"]`,
      parent: this,
    });
  }
  // missing templates, archived, sort, list view button, list view button, list view, new project button
}

/**
 * Represents the ProjectsCard in the WorkspaceProjects component
 */
class ProjectsCard extends Card {
  override readonly actionMenu = new ProjectActionDropdown({
    clickThisComponentToOpen: this.actionMenuContainer,
    root: this.root,
  });
}
