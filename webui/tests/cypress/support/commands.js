// ***********************************************
// This example commands.js shows you how to
// create various custom commands and overwrite
// existing commands.
//
// For more comprehensive examples of custom
// commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************
//
//
// -- This is a parent command --
// Cypress.Commands.add('login', () => { cy.visit('/'); });
//
//
// -- This is a child command --
// Cypress.Commands.add("drag", { prevSubject: 'element'}, (subject, options) => { ... })
//
//
// -- This is a dual command --
// Cypress.Commands.add("dismiss", { prevSubject: 'optional'}, (subject, options) => { ... })
//
//
// -- This will overwrite an existing command --
// Cypress.Commands.overwrite("visit", (originalFn, url, options) => { ... })

Cypress.Commands.add('dataCy', (value) => {
  return cy.get(`[data-test=${value}]`);
});

Cypress.Commands.add('checkLoggedIn', username => {
  // Check for the presence/absence of the icons for the user dropdown and
  // cluster page link in the top bar, which should be present if and only if
  // the user is logged in.
  cy.get('#avatar').should('exist');
  cy.get('#avatar').should('have.text', username.charAt(0).toUpperCase());
});

Cypress.Commands.add('checkLoggedOut', () => {
  cy.get('#avatar').should('not.exist');
});

Cypress.Commands.add('login', credentials => {
  cy.request('POST', '/login', credentials);
  cy.checkLoggedIn(credentials.username);
});

Cypress.Commands.add('logout', () => {
  cy.request('POST', '/logout');
  cy.checkLoggedOut();
});
