/// <reference types="cypress" />

declare namespace Cypress {
  interface Chainable<Subject> {
    /**
     * Make a request to log in and check.
     * @example
     * cy.login({username: determined})
     */
    login(credentials: { username: string; password?: string }): Chainable<any>;
    /**
     * Make a request to log out and check.
     * @example
     * cy.logout()
     */
    logout(): Chainable<any>;
    /**
     * Check that the application is in a logged in state
     * @example
     * cy.checkLoggedIn()
     */
    checkLoggedIn(username: string): Chainable<any>;
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
  }
}
