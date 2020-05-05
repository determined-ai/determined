// ***********************************************
// For examples of custom commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************

const DEFAULT_TEST_USER = 'determined';

Cypress.Commands.add('dataCy', (value) => {
  return cy.get(`[data-test=${value}]`);
});

Cypress.Commands.add('checkLoggedIn', username => {
  // Check for the presence/absence of the icons for the user dropdown and
  // cluster page link in the top bar, which should be present if and only if
  // the user is logged in.
  username = username || DEFAULT_TEST_USER;
  cy.visit('/');
  cy.get('#avatar').should('exist');
  cy.get('#avatar').should('have.text', username.charAt(0).toUpperCase());
});

Cypress.Commands.add('checkLoggedOut', () => {
  cy.visit('/');
  cy.get('#avatar').should('not.exist');
});

// TODO use Cypress.env to share (and bring in) some of the contants used.
Cypress.Commands.add('login', credentials => {
  credentials = credentials || { username: DEFAULT_TEST_USER };
  cy.request('POST', '/login', credentials)
    .then(response => {
      expect(response.body).to.have.property('token');
      return cy.setCookie('auth', response.body.token);
    });
  cy.getCookie('auth')
    .should('exist')
    .should('have.property', 'value');
  cy.checkLoggedIn(credentials.username);
});

Cypress.Commands.add('logout', () => {
  cy.request({
    failOnStatusCode: false, // make this command idempotent
    method: 'POST',
    url: '/logout',
  })
    .then(() => {
      return cy.clearCookie('auth');
    });
  cy.checkLoggedOut();
});
