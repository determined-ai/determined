import { Locator, type Page } from '@playwright/test';

import { Navigation } from 'e2e/models/components/Navigation';

export interface ModelBasics {
  get pwLocator(): Locator;
}

/**
 * Base model for any Page in src/pages/
 */
export abstract class BasePage implements ModelBasics {
  readonly _page: Page;
  readonly nav = new Navigation({ parent: this });

  abstract readonly url: string | RegExp;
  abstract readonly title: string | RegExp;

  /**
   * Constructs a BasePage
   * @param {Page} page - Playwright's Page object
   */
  constructor(page: Page) {
    this._page = page;
  }

  /**
   * The playwright top-level locator
   */
  get pwLocator(): Locator {
    return this._page.locator(':root');
  }

  /**
   * Returns this so we can chain. Visits the page.
   * @param {{}} [args] - obj
   * @param {string} args.url - A URL to visit. It can be different from the URL to verify
   * @param {boolean} [args.verify] - Whether for the URL to change
   */
  async goto({
    url = this.url,
    verify = true,
  }: { url?: string | RegExp; verify?: boolean } = {}): Promise<this> {
    if (url instanceof RegExp) {
      throw new Error(`${typeof this}.url is a regular expression. Please provide a url to visit.`);
    }
    await this._page.goto(url);
    if (verify) {
      // TODO this does nothing because we end up checking, passing, and then getting redirected
      await this._page.waitForURL(this.url);
    }
    return this;
  }

  async waitForURL(): Promise<void> {
    if (this.url instanceof RegExp) await this._page.waitForURL(this.url);
    await this._page.waitForURL(new RegExp(this.url));
  }

  /**
   * Logs a string to the browser console. This string will show in playwright's trace.
   * @param {string} s - the string to log to the browser console
   */
  async browserLog(s: string): Promise<void> {
    await this._page.evaluate((s: string) => {
      // eslint-disable-next-line no-console
      console.log(s);
    }, s);
  }
}
