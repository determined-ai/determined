describe('Sign in/out', () => {

  const LOGIN_ROUTE = '/det/login';
  const LOGOUT_ROUTE = '/det/logout';
  const elmTitleSelector = '#det-main-container div.text-2xl';

  it('should be logged in React side', () => {
    cy.login();
    cy.visit('/det');
    cy.checkLoggedIn();
  });

  it('should be logged in Elm side', () => {
    cy.login();
    cy.visit('/ui');
    cy.checkLoggedIn();
  });

  it('should be able to log out from Elm', () => {
    // Open the dropdown menu and click the button to log out.
    cy.login();
    cy.visit('/ui');
    cy.get('#avatar').click();
    cy.get(`nav a[href="${LOGOUT_ROUTE}"]`).should('have.lengthOf', 1);
    cy.get(`nav a[href="${LOGOUT_ROUTE}"]`).click();
    cy.checkLoggedOut();
  });

  it('should be able to log out from React', () => {
    // Open the dropdown menu and click the button to log out.
    cy.login();
    cy.visit('/det/dashboard');
    cy.get('#avatar').click();
    // TODO add better identifiers to react Link component. make it an anchor tag?
    cy.get(`[role="menu"] a[href="${LOGOUT_ROUTE}"]`).should('have.lengthOf', 1);
    cy.get(`[role="menu"] a[href="${LOGOUT_ROUTE}"]`).click();
    cy.checkLoggedOut();
  });

  it('should be able to log in', () => {
    const username = 'determined';

    cy.logout();
    cy.visit(LOGIN_ROUTE);

    cy.get('input#login_username')
      .type(username, { delay: 100 })
      .should('have.value', username);

    cy.server();
    cy.route('POST', /\/login.*/).as('loginRequest');
    cy.get('button[type="submit"]').click();
    cy.wait('@loginRequest');
    cy.checkLoggedIn(username);
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

  it('should redirect to the requested elm url after login', () => {
    cy.login();
    cy.visit(`${LOGIN_ROUTE}?redirect=/ui/experiments`);
    cy.get(elmTitleSelector).contains('Experiments');
  });
});
