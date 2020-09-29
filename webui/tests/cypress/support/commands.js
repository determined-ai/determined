// ***********************************************
// For examples of custom commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************
import { ACCOUNT_PASSWORD, ACCOUNT_USERNAME, API_PATH } from '../constants';

const sha512 = require('js-sha512').sha512;

const saltAndHashPassword = password => {
  if (!password) return '';
  const passwordSalt = 'GubPEmmotfiK9TMD6Zdw';
  return sha512(passwordSalt + password);
};

Cypress.on('uncaught:exception', () => {
  // returning false here prevents Cypress from failing the test
  return false;
});

Cypress.Commands.add('dataCy', (value) => {
  return cy.get(`[data-test=${value}]`);
});

Cypress.Commands.add('checkLoggedIn', (username = null, visit = true) => {
  // Check for the presence/absence of the icons for the user dropdown and
  // cluster page link in the top bar, which should be present if and only if
  // the user is logged in.
  username = username || ACCOUNT_USERNAME;
  if (visit) cy.visit('/');
  cy.get('#avatar').should('exist');
  cy.get('#avatar').should('have.text', username.charAt(0).toUpperCase());
});

Cypress.Commands.add('checkLoggedOut', () => {
  cy.visit('/');
  cy.request({
    failOnStatusCode: false,
    method: 'GET',
    url: '/users/me',
  })
    .then(response => {
      expect(response.status).to.equal(401);
    });
});

// TODO use Cypress.env to share (and bring in) some of the contants used.
Cypress.Commands.add('login', credentials => {
  credentials = credentials || {
    password: saltAndHashPassword(ACCOUNT_PASSWORD),
    username: ACCOUNT_USERNAME,
  };
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
    url: `${API_PATH}/auth/logout`,
  })
    .then(() => {
      return cy.clearCookie('auth');
    });
  cy.checkLoggedOut();
});
