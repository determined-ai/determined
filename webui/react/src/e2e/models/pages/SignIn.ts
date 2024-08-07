import { DeterminedPage } from 'e2e/models/common/base/BasePage';
import { DeterminedAuth } from 'e2e/models/components/DeterminedAuth';
import { PageComponent } from 'e2e/models/components/Page';

/**
 * Represents the SignIn page from src/pages/SignIn.tsx
 */
export class SignIn extends DeterminedPage {
  readonly title = 'Sign In';
  readonly url = 'login';
  readonly pageComponent = new PageComponent({ parent: this });
  readonly detAuth = new DeterminedAuth({ parent: this.pageComponent });
  // TODO add SSO page model as well
}
