import { BasePage } from 'e2e/models/BasePage';
import { WorkspacesList } from 'e2e/models/components/WorkspacesList';
import { WorkspaceDetails } from 'e2e/models/components/WorkspaceDetails';


/**
 * Returns a representation of an Workspaces page.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export class Workspaces extends BasePage {
  readonly title: string = 'Workspaces - Determined';
  readonly url: string = 'workspaces';
  readonly list: WorkspacesList = new WorkspacesList({ 
    parent:this 
  })
  readonly projects: WorkspaceDetails = new WorkspaceDetails({ 
    parent:this
  })
}
