/// <reference types="cypress" />

declare namespace Cypress {
  interface Chainable<Subject> {
    /**
     * Log the user in by driving the UI.
     * @example
     * cy.login({username: determined})
     */
    login(credentials?: { username: string; password?: string }): Chainable<any>;
    /**
     * Make a request to log in and check.
     * @example
     * cy.login({username: determined})
     */
    loginHeadless(credentials?: { username: string; password?: string }): Chainable<any>;
    /**
     * Make a request to log out and check.
     * @example
     * cy.logout()
     */
    // logout(): Chainable<any>;
    /**
     * Check that the application is in a logged in state
     * @example
     * cy.checkLoggedIn()
     */
    checkLoggedIn(username?: string, visit?: boolean): Chainable<any>;
    /**
     * Check that the application is in a logged out state
     * @example
     * cy.checkLoggedOut()
     */
    checkLoggedOut(): Chainable<any>;
    /**
     * Custom command to select DOM element by data-cy attribute.
     * @example cy.dataCy('greeting')
    */
    dataCy(value: string): Chainable<Element>;
    /**
     * save local storage.
    */
    saveLocalStorageCache(keys?: string[]): Chainable<Element>;
    /**
     * restore a previously saved local storage.
    */
    restoreLocalStorageCache(keys?: string[]): Chainable<Element>;
  }
}
