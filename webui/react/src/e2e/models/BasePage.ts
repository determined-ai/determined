import { type Page } from '@playwright/test';

/**
 * Returns the representation of a Page.
 *
 * @remarks
 * This constructor is a base class for any component in src/pages/.
 *
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export abstract class BasePage {

  readonly _page: Page;
  readonly url: string | undefined;

  constructor(page: Page) {
    this._page = page;
  }

  /**
   * The playwright locator method from this model's page
   */
  get pwLocatorFunction() { return this._page.locator }

  /**
   * Returns this so we can chain.
   *
   * @remarks
   * ie. await expect(thePage.goto().theElement.loc()).toBeVisible()
   *
   * @param {Page} [waitForURL] - Whether for the URL to change
   */
  goto(waitForURL: boolean = true): BasePage {
    if (typeof this.url === "undefined") {
      throw new Error('URL is not set');
    }
    this._page.goto(this.url);
    if (waitForURL) {
      this._page.waitForURL(this.url);
    }
    return this
  }

}
