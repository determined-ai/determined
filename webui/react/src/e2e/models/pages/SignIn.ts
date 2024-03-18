import { type Page } from '@playwright/test';
import { DeterminedAuth } from '../components/DeterminedAuth'
import { BasePage } from '../BasePage'
import { Page as PageComponent } from '../components/Page'

export class SignIn extends BasePage {
  readonly page: PageComponent;

  constructor(page: Page) {
    super(page)
    this.page = new PageComponent({parent: this, subelements:[
      {name: 'determinedAuth', type: DeterminedAuth, selector: DeterminedAuth.defaultSelector}
    ]})
  }
}