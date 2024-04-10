import { Locator } from "@playwright/test";
import { BasePage } from "e2e/models/BasePage";

/**
 * Returns a representation of a modal that is detached from the :root of the initial page.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export class ModalRootPage extends BasePage {
    override url: string = '';
    override title: string = '';
    override get pwLocator(): Locator {
        return this._page.locator('div.ant-modal-root');
    }
}