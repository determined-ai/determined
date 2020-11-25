import { DEFAULT_WAIT_TIME, LOGIN_ROUTE, LOGOUT_ROUTE, USERNAME_INPUT,
  USERNAME_WITHOUT_PASSWORD } from '../constants';

describe('Sign in/out', () => {

  const logoutSelector = `[class^="Navigation_base_"] [role="menu"] a[href*="${LOGOUT_ROUTE}"]`;

  it('should be logged out', () => {
    cy.checkLoggedOut();
  });

  it('should be able to log out from React', () => {
    // Open the dropdown menu and click the button to log out.
    cy.login();
    cy.visit('/det');
    cy.get('#avatar').click();
    cy.get(logoutSelector).click();
    cy.checkLoggedOut();
  });

  it('should be able to log in', () => {
    cy.visit(LOGIN_ROUTE);
    cy.get(USERNAME_INPUT).should('have.value', '');
    cy.get(USERNAME_INPUT)
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

  it('should stay logged in after reload', () => {
    cy.login();
    cy.visit('/det');
    cy.reload();
    cy.checkLoggedIn();
  });

  it('should redirect away from login when visiting login while logged in', () => {
    cy.login();
    cy.visit(LOGIN_ROUTE);
    cy.url().should('not.contain', LOGIN_ROUTE);
  });

  it('should logout the user when visiting the logout page', () => {
    cy.login();
    cy.checkLoggedIn();
    cy.visit(LOGOUT_ROUTE);
    cy.checkLoggedOut();
  });

  it('should end up redirecting to login page when visiting logout page', () => {
    cy.visit(LOGOUT_ROUTE);
    cy.url().should('contain', LOGIN_ROUTE);
    cy.login();
    cy.visit(LOGOUT_ROUTE);
    cy.url().should('contain', LOGIN_ROUTE);
  });
});
