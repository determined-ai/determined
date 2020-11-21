import { DEFAULT_WAIT_TIME, USERNAME_WITHOUT_PASSWORD } from '../constants';

describe('Sign in/out', () => {

  const LOGIN_ROUTE = '/det/login';
  const LOGOUT_ROUTE = '/det/logout';
  const logoutSelector = `[class^="Navigation_base_"] [role="menu"] a[href*="${LOGOUT_ROUTE}"]`;

  before(() => {
    cy.login();
  });

  after(() => {
    cy.login();
  });

  it('should be logged in React side', () => {
    cy.visit('/det');
    cy.checkLoggedIn();
  });

  it('should be able to log out from React', () => {
    // Open the dropdown menu and click the button to log out.
    cy.login();
    cy.visit('/det/dashboard');
    cy.get('#avatar').click();
    cy.get(logoutSelector).click();
    cy.checkLoggedOut();
  });

  it('should be able to log in', () => {
    cy.logout();
    cy.visit(LOGIN_ROUTE);

    cy.get('input#login_username')
      .type(USERNAME_WITHOUT_PASSWORD, { delay: 100 })
      .should('have.value', USERNAME_WITHOUT_PASSWORD);

    cy.get('button[type="submit"]').contains('Sign In').click();

    /*
     * Cypress is unable to capture /api/v1/auth/login POST requests properly
     * via `cy.route`, instead having to rely on a time-based wait.
     * https://github.com/cypress-io/cypress/issues/2188
     */
    /* eslint-disable-next-line cypress/no-unnecessary-waiting */
    cy.wait(DEFAULT_WAIT_TIME);
    cy.checkLoggedIn(USERNAME_WITHOUT_PASSWORD, false);
  });

  it('should redirect away from login when visiting login while logged in', () => {
    cy.login();
    cy.visit(LOGIN_ROUTE);
    cy.url().should('not.contain', LOGIN_ROUTE);
  });

  it('should logout the user when visiting the logout page', () => {
    cy.login();
    cy.visit(LOGOUT_ROUTE);
    cy.checkLoggedOut();
  });

  it('should end up redirecting to login page when visiting logout page', () => {
    cy.visit(LOGOUT_ROUTE);
    cy.url().should('contain', LOGIN_ROUTE);
  });
});
