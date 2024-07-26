import { BaseComponent } from 'e2e/models/common/base/BaseComponent';
import { BasePage } from 'e2e/models/common/base/BasePage';
import { Message } from 'e2e/models/common/hew/Message';
import { Select } from 'e2e/models/common/hew/Select';
import { Toggle } from 'e2e/models/common/hew/Toggle';
import { WorkspaceCreateModal } from 'e2e/models/components/WorkspaceCreateModal';
import { WorkspaceDeleteModal } from 'e2e/models/components/WorkspaceDeleteModal';
import { WorkspaceCard } from 'e2e/models/pages/WorkspaceList/WorkspaceCard';

/**
 * Represents the WorkspaceList page from src/pages/WorkspaceList.tsx
 */
export class WorkspaceList extends BasePage {
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
  // TODO missing grid list
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
