import { type Page } from '@playwright/test';

import { BasePage } from 'e2e/models/BasePage';
import { DeterminedAuth } from 'e2e/models/components/DeterminedAuth';
import { Page as PageComponent } from 'e2e/models/components/Page';

export class SignIn extends BasePage {
  readonly page: PageComponent;

  constructor(page: Page) {
    super(page);
    this.page = new PageComponent({
      parent: this,
subelements: [
        { name: 'determinedAuth', selector: DeterminedAuth.defaultSelector, type: DeterminedAuth },
      ],
    });
  }
}
