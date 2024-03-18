import { type Page } from '@playwright/test';

import { hasSubelements } from './BaseComponent';

export class BasePage extends hasSubelements {
    readonly _page: Page;
    readonly url: string | null = null;

    constructor(page: Page) {
        super();
        this._page = page;
    }

    visit(waitFor: boolean = true): void {
        if (this.url == null) {
            throw new Error('URL is not set');
        }
        this._page.goto(this.url);
        if (waitFor) {
            this._page.waitForURL(this.url);
        }
    }

    override locate(): Page {
        return this._page;
    }
}
