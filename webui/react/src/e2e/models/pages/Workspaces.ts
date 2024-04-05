import { BasePage } from 'e2e/models/BasePage';


/**
 * Returns a representation of an Workspaces page.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export class Workspaces extends BasePage {
    readonly title: string = 'Workspaces - Determined';
  readonly url: string = 'workspaces';
}
