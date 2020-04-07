/// <reference types="cypress" />

declare namespace Cypress {
  interface Chainable<Subject> {
    /**
     * Ensure auto login is triggered and user is logged in.
     * @example
     * cy.login()
     */
    login(): Chainable<any>;
    /**
     * Custom command to select DOM element by data-cy attribute.
     * @example cy.dataCy('greeting')
    */
    dataCy(value: string): Chainable<Element>;
  }
}
