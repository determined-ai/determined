import { BasePage as BP } from 'playwright-page-model-base/BasePage';

import { Navigation } from 'e2e/models/components/Navigation';

/**
 * Base model for any Page in src/pages/
 */
export abstract class BasePage extends BP {
  readonly nav = new Navigation({ parent: this });
}
