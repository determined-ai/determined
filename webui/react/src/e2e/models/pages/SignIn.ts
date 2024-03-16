import { type Page } from '@playwright/test';
import { DeterminedAuth } from '../components/DeterminedAuth'

export class SignIn {
  readonly page: Page;
  readonly determinedAuth: DeterminedAuth;

  constructor(page: Page) {
    this.page = page;
    this.determinedAuth = new DeterminedAuth(page.getByTestId(DeterminedAuth.defaultLocator));
  }
}