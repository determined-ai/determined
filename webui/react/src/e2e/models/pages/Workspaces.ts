import { BasePage } from 'e2e/models/BasePage';
import { WorkspaceCreateModal } from 'e2e/models/components/WorkspaceCreateModal';
import { WorkspaceDeleteModal } from 'e2e/models/components/WorkspaceDeleteModal';
import { WorkspaceDetails } from 'e2e/models/components/WorkspaceDetails';
import { WorkspacesList } from 'e2e/models/components/WorkspacesList';

/**
 * Returns a representation of an Workspaces page.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export class Workspaces extends BasePage {
  readonly title: string = 'Workspaces - Determined';
  readonly url: string = 'workspaces';
  readonly list: WorkspacesList = new WorkspacesList({
    parent: this,
  });
  readonly projects: WorkspaceDetails = new WorkspaceDetails({
    parent: this,
  });
  readonly createModal: WorkspaceCreateModal = new WorkspaceCreateModal({
    parent: this,
  });
  readonly deleteModal: WorkspaceDeleteModal = new WorkspaceDeleteModal({
    parent: this,
  });
}
