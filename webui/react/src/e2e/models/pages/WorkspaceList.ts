import { BaseComponent } from 'e2e/models/common/base/BaseComponent';
import { BasePage } from 'e2e/models/common/base/BasePage';
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
  readonly newWorkspaceButton = new BaseComponent({
    parent: this,
    selector: '[data-testid="newWorkspace"]',
  });
  readonly workspaceCards = new WorkspaceCard({
    parent: this,
  });
  cardByName(name: string): WorkspaceCard {
    return new WorkspaceCard({
      attachment: `[data-testid="card-${name}"]`,
      parent: this,
    });
  }
  // missing stuff like workspace select, archived, sort, list view button, list view
}
