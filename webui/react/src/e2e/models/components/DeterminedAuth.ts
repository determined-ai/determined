import { type Locator } from '@playwright/test';

export class DeterminedAuth {
    readonly locator: Locator;
    static defaultLocator: string = 'a';
    readonly username: Locator
    readonly password: Locator
    readonly docs: Locator

    constructor(locator: Locator) {
        this.locator = locator;
        this.username = this.locator.getByTestId('username')
        this.password = this.locator.getByTestId('password')
        this.docs = this.locator.getByRole('link')
    }
}