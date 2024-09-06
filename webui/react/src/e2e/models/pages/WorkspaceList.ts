import { BaseComponent } from 'playwright-page-model-base/BaseComponent';

import { DeterminedPage } from 'e2e/models/common/base/BasePage';
import { Message } from 'e2e/models/common/hew/Message';
import { Select } from 'e2e/models/common/hew/Select';
import { Toggle } from 'e2e/models/common/hew/Toggle';
import { GridListRadioGroup } from 'e2e/models/components/GridListRadioGroup';
import { HeadRow, InteractiveTable, Row } from 'e2e/models/components/Table/InteractiveTable';
import { WorkspaceCreateModal } from 'e2e/models/components/WorkspaceCreateModal';
import { WorkspaceDeleteModal } from 'e2e/models/components/WorkspaceDeleteModal';
import { WorkspaceCard } from 'e2e/models/pages/WorkspaceList/WorkspaceCard';

class WorkspaceHeadRow extends HeadRow {
  readonly name = new BaseComponent({
    parent: this,
    selector: '[data-testid="name"]',
  });
}

class WorkspaceRow extends Row {
  readonly name = new BaseComponent({
    parent: this,
    selector: '[data-testid="name"]',
  });
}

/**
 * Represents the WorkspaceList page from src/pages/WorkspaceList.tsx
 */
export class WorkspaceList extends DeterminedPage {
  readonly title = 'Workspaces';
  readonly url = 'workspaces';
  readonly createModal = new WorkspaceCreateModal({
    root: this,
  });
  readonly deleteModal = new WorkspaceDeleteModal({
    root: this,
  });
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
  readonly newWorkspaceButton = new BaseComponent({
    parent: this,
    selector: '[data-testid="newWorkspace"]',
  });
  readonly gridListRadioGroup = new GridListRadioGroup({
    parent: this,
    selector: '[data-testid="grid-list-radio-group"]',
  });
  readonly table = new InteractiveTable({
    parent: this,
    tableArgs: {
      attachment: '[data-testid="table"]',
      headRowType: WorkspaceHeadRow,
      rowType: WorkspaceRow,
    },
  });
  readonly workspaceCards = new WorkspaceCard({
    parent: this,
  });
  readonly noWorkspacesMessage = new Message({
    parent: this,
    selector: '[data-testid="noWorkspaces"]',
  });
  readonly noMatchingWorkspacesMessage = new Message({
    parent: this,
    selector: '[data-testid="noMatchingWorkspaces"]',
  });
  cardByName(name: string): WorkspaceCard {
    return new WorkspaceCard({
      attachment: `[data-testid="card-${name}"]`,
      parent: this,
    });
  }
}
