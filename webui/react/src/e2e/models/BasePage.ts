import { type Page } from '@playwright/test';

import { canBeParent } from './BaseComponent';

export abstract class BasePage extends canBeParent {
  readonly #page: Page;
  readonly url: string | null = null;

  /**
   * Returns the representation of a Page.
   *
   * @remarks
   * This constructor is a base class for any component in src/pages/.
   *
   * @param {Page} page - The '@playwright/test' Page being used by a test
   */
  constructor(page: Page) {
    super();
    this.#page = page;
  }

  /**
   * Returns this so we can chain.
   * ie. await expect(thePage.goto().theElement.loc()).toBeVisible()
   *
   * @remarks
   * This constructor is a base class for any component in src/pages/.
   *
   * @param {Page} page - The '@playwright/test' Page being used by a test
   */
  goto(waitFor: boolean = true): BasePage {
    if (this.url == null) {
      throw new Error('URL is not set');
    }
    this.#page.goto(this.url);
    if (waitFor) {
      this.#page.waitForURL(this.url);
    }
    return this
  }

  /**
   * Returns this object's Page.
   *
   * @remarks
   * We use this method to call this.loc.locate().
   */
  override get locator(): Page {
    return this.#page;
  }
}
