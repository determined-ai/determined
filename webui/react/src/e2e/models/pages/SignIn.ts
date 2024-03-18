import { type Page } from '@playwright/test';

import { BasePage } from 'e2e/models/BasePage';
import { DeterminedAuth } from 'e2e/models/components/DeterminedAuth';
import { Page as PageComponent } from 'e2e/models/components/Page';

export class SignIn extends BasePage {
  readonly page: PageComponent;

  /**
   * Returns a representation of the SignIn page.
   *
   * @remarks
   * This constructor represents the contents in src/pages/SignIn.tsx.
   *
   * @param {Page} page - The '@playwright/test' Page being used by a test
   */
  constructor(page: Page) {
    super(page);
    this.page = new PageComponent({
      parent: this,
      subComponents: [
        { name: 'determinedAuth', selector: DeterminedAuth.defaultSelector, type: DeterminedAuth },
      ],
    });
    // TODO add SSO page model as well
  }
}
