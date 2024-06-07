import { BasePage } from 'e2e/models/BasePage';
import { DeterminedAuth } from 'e2e/models/components/DeterminedAuth';
import { PageComponent } from 'e2e/models/components/Page';

/**
 * Represents the SignIn page from src/pages/SignIn.tsx
 */
export class SignIn extends BasePage {
  readonly title: string = SignIn.getTitle('Sign In');
  readonly url: string = 'login';
  readonly pageComponent = new PageComponent({ parent: this });
  readonly detAuth = new DeterminedAuth({ parent: this.pageComponent });
  // TODO add SSO page model as well
}
