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

class ProjectHeadRow extends HeadRow {
  readonly name = new BaseComponent({
    parent: this,
    selector: '[data-testid="Name"]',
  });
  readonly description = new BaseComponent({
    parent: this,
    selector: '[data-testid="Description"]',
  });
  readonly numExperiments = new BaseComponent({
    parent: this,
    selector: '[data-testid="NumExperiments"]',
  });
  readonly lastUpdated = new BaseComponent({
    parent: this,
    selector: '[data-testid="LastUpdated"]',
  });
  readonly userId = new BaseComponent({
    parent: this,
    selector: '[data-testid="UserId"]',
  });
  readonly archived = new BaseComponent({
    parent: this,
    selector: '[data-testid="Archived"]',
  });
  readonly state = new BaseComponent({
    parent: this,
    selector: '[data-testid="State"]',
  });
  readonly action = new BaseComponent({
    parent: this,
    selector: '[data-testid="Action"]',
  });
}

class ProjectRow extends Row {
  readonly name = new BaseComponent({
    parent: this,
    selector: '[data-testid="name"]',
  });
  readonly description = new BaseComponent({
    parent: this,
    selector: '[data-testid="Description"]',
  });
  readonly numExperiments = new BaseComponent({
    parent: this,
    selector: '[data-testid="NumExperiments"]',
  });
  readonly lastUpdated = new BaseComponent({
    parent: this,
    selector: '[data-testid="LastUpdated"]',
  });
  readonly userId = new BaseComponent({
    parent: this,
    selector: '[data-testid="UserId"]',
  });
  readonly archived = new BaseComponent({
    parent: this,
    selector: '[data-testid="Archived"]',
  });
  readonly state = new BaseComponent({
    parent: this,
    selector: '[data-testid="State"]',
  });
  readonly action = new BaseComponent({
    parent: this,
    selector: '[data-testid="Action"]',
  });
}

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
