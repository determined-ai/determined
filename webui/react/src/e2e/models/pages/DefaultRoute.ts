import { DeterminedPage } from 'e2e/models/common/base/BasePage';
import { isRbacEnabled } from 'e2e/utils/rbac';

/**
 * Represents the DefaultRoute page from src/pages/DefaultRoute.tsx
 */
export class DefaultRoute extends DeterminedPage {
  // only redirects to Dashboard or WorkspaceList page
  readonly title = isRbacEnabled() ? 'Workspaces' : 'Home';
  readonly url = isRbacEnabled() ? /workspaces/ : /dashboard/;
}
