import { BaseComponent } from 'playwright-page-model-base/BaseComponent';
import { BaseReactFragment } from 'playwright-page-model-base/BaseReactFragment';

import { Select } from 'e2e/models/common/hew/Select';
import { Toggle } from 'e2e/models/common/hew/Toggle';
import { GridListRadioGroup } from 'e2e/models/components/GridListRadioGroup';
import { ProjectCard } from 'e2e/models/components/ProjectCard';
import { ProjectCreateModal } from 'e2e/models/components/ProjectCreateModal';
import { ProjectDeleteModal } from 'e2e/models/components/ProjectDeleteModal';
import { ProjectMoveModal } from 'e2e/models/components/ProjectMoveModal';
import { HeadRow, InteractiveTable, Row } from 'e2e/models/components/Table/InteractiveTable';

// import { ProjectActionDropdown } from 'e2e/models/components/ProjectActionDropdown';

class ProjectHeadRow extends HeadRow {
  readonly name = new BaseComponent({
    parent: this,
    selector: '[data-testid="Name"]',
  });
}

class ProjectRow extends Row {
  readonly name = new BaseComponent({
    parent: this,
    selector: '[data-testid="name"]',
    // TODO: add all columns from WorkspaceProjects file here
    // TODO: make sure to have the variable to open
  });
}
// readonly actionMenuContainer = new BaseComponent({
//   parent: this,
//   selector: '[aria-label="Action menu"]',
// });
//   readonly actionMenu = new ProjectActionDropdown({
//   clickThisComponentToOpen: this.actionMenuContainer,
//   root: this.root, // root of the dropdown
// });

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
  readonly gridListRadioGroup = new GridListRadioGroup({
    parent: this,
  });
  readonly table = new InteractiveTable({
    parent: this,
    tableArgs: {
      attachment: '[data-testid="table"]',
      headRowType: ProjectHeadRow,
      rowType: ProjectRow,
    },
  });
  readonly projectCards = new ProjectCard({
    parent: this,
  });
  readonly createModal = new ProjectCreateModal({
    root: this.root,
  });
  readonly deleteModal = new ProjectDeleteModal({
    root: this.root,
  });
  readonly moveModal = new ProjectMoveModal({
    root: this.root,
  });
  cardByName(name: string): ProjectCard {
    return new ProjectCard({
      attachment: `[data-testid="card-${name}"]`,
      parent: this,
    });
  }
}
