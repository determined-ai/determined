import { expect, Locator, type Page } from '@playwright/test';

import { Navigation } from 'e2e/models/components/Navigation';

export interface ModelBasics {
  get pwLocator(): Locator;
}

/**
 * Returns the representation of a Page.
 * This constructor is a base class for any component in src/pages/.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export abstract class BasePage implements ModelBasics {
  readonly _page: Page;
  readonly nav: Navigation = new Navigation({ parent: this });

  private static isEE = Boolean(JSON.parse(process.env.PW_EE ?? ''));
  private static appTitle = BasePage.isEE
    ? 'HPE Machine Learning Development Environment'
    : 'Determined';

  abstract readonly url: string | RegExp;
  abstract readonly title: string | RegExp;

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
   * The title of the page. Format of [prefix - ]appTitle
   * @param prefix
   */
  public static getTitle(prefix: string = ''): string {
    if (prefix === '') {
      return BasePage.appTitle;
    }
    return `${prefix} - ${BasePage.appTitle}`;
  }

  public getUrlRegExp(): RegExp {
    if (this.url instanceof RegExp) {
      return this.url;
    }
    return new RegExp(this.url, 'g');
  }

  /**
   * Returns this so we can chain. Visits the page.
   * ie. await expect(thePage.goto().theElement.pwLocator()).toBeVisible()
   * @param {{}} [args] - obj
   * @param {string} args.url - A URL to visit. It can be different from the URL to verify
   * @param {boolean} [args.verify] - Whether for the URL to change
   */
  async goto(
    { url, verify = true }: { url: string | RegExp; verify?: boolean } = this,
  ): Promise<BasePage> {
    if (url instanceof RegExp) {
      throw new Error(`${typeof this}.url is a regular expression. Please provide a url to visit.`);
    }
    await this._page.goto(url);
    if (verify) {
      await this._page.waitForURL(this.url);
      await expect(this._page).toHaveTitle(this.title);
    }
    return this;
  }
}
