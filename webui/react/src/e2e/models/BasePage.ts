import { Locator, type Page } from '@playwright/test';

import { requireStaticArgs } from 'e2e/models/BaseComponent';
import { Navigation } from 'e2e/models/components/Navigation';

/**
 * Returns the representation of a Page.
 * This constructor is a base class for any component in src/pages/.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export abstract class BasePage {
  readonly _page: Page;
  readonly nav: Navigation = new Navigation({ parent: this });

  constructor(page: Page) {
    this._page = page;
    requireStaticArgs(this.constructor, ['url', 'title']);
  }

  /**
   * The playwright top-level locator
   */
  get pwLocator(): Locator {
    return this._page.locator(':root');
  }

  /**
   * Returns this so we can chain.
   * ie. await expect(thePage.goto().theElement.loc()).toBeVisible()
   * @param {Page} [waitForURL] - Whether for the URL to change
   */
  goto(waitForURL: boolean = true): BasePage {
    const url: string = Object.getPrototypeOf(this).url;
    this._page.goto(url);
    if (waitForURL) {
      this._page.waitForURL(url);
    }
    return this;
  }
}
