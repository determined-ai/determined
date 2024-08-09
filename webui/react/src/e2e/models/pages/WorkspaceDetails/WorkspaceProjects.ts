import { BaseComponent } from 'playwright-page-model-base/BaseComponent';
import { BaseReactFragment } from 'playwright-page-model-base/BaseReactFragment';

import { Select } from 'e2e/models/common/hew/Select';
import { Toggle } from 'e2e/models/common/hew/Toggle';
import { ProjectsCard } from 'e2e/models/components/ProjectCard';
import { ProjectCreateModal } from 'e2e/models/components/ProjectCreateModal';
import { ProjectDeleteModal } from 'e2e/models/components/ProjectDeleteModal';

/**
 * Represents the WorkspaceProjects page in src/pages/WorkspaceDetails/WorkspaceProjects.tsx
 */
export class WorkspaceProjects extends BaseReactFragment {
  readonly tab = 'projects';
  readonly url = /workspaces\/\d+\/projects/;
  readonly whoseSelect = new Select({
    parent: this,
    selector: '[data-testid="whose"]',
  });
  readonly showArchived = new Toggle({
    parent: this,
    selector: '[class^="Column"] [class^="Row"] [class^="Row"]',
  });
  readonly sortSelect = new Select({
    parent: this,
    selector: '[data-testid="sort"]',
  });
  readonly newProject = new BaseComponent({
    parent: this,
    selector: '[data-testid="newProject"]',
  });
  // TODO missing grid toggle
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
}
