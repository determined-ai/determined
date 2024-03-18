import { type Page } from '@playwright/test';
import { BaseComponent, Subelement } from './BaseComponent';


export class BasePage {
    readonly _page: Page;
    readonly url: string | null = null;

    constructor(page: Page) {
        this._page = page;
    }

    _initialize_subelements(subelements: Subelement[]) {
        subelements.forEach(subelement => {
            Object.defineProperty(this, subelement.name, new BaseComponent({
                parent: this,
                selector: subelement.selector,
                subelements: subelement.subelements
            }))
        });
    }

    visit(waitFor: boolean = true) {
        if (this.url == null) {
            throw new Error(`URL is not set`);
        }
        this._page.goto(this.url)
        if (waitFor) {
            this._page.waitForURL(this.url)
        }
    }

    locate() {
        return this._page
    }
}