import { BasePage } from 'e2e/models/base/BasePage';
import { WorkspaceCreateModal } from 'e2e/models/components/WorkspaceCreateModal';
import { WorkspaceDeleteModal } from 'e2e/models/components/WorkspaceDeleteModal';
import { WorkspaceDetails } from 'e2e/models/components/WorkspaceDetails';
import { WorkspacesList } from 'e2e/models/components/WorkspacesList';

/**
 * Represents the Workspaces page from pages/WorkspacesList.tsx
 * Represents the Workspaces page from pages/WorkspaceDetails.tsx
 * Represents the Workspaces page from pages/WorkspaceCreateModal.tsx
 * Represents the Workspaces page from pages/WorkspaceDeleteModal.tsx
 */
export class Workspaces extends BasePage {
  readonly title: string = Workspaces.getTitle('Workspaces');
  readonly url: string = 'workspaces';
  readonly list = new WorkspacesList({
    parent: this,
  });
  readonly details = new WorkspaceDetails({
    parent: this,
  });
  readonly createModal = new WorkspaceCreateModal({
    root: this,
  });
  readonly deleteModal = new WorkspaceDeleteModal({
    root: this,
  });
}
