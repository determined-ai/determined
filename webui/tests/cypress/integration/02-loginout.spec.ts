describe('Sign in/out', () => {

  const LOGIN_ROUTE = '/det/login';
  const LOGOUT_ROUTE = '/det/logout';

  before(() => {
    cy.login();
  });

  const elmTitleSelector = '#det-main-container div.text-2xl';

  it('should be logged in React side', () => {
    cy.visit('/det/dashboard');
    cy.checkLoggedIn();
  });

  it('should be logged in Elm side', () => {
    cy.visit('/ui/experiments');
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
    cy.visit('/det');
    cy.get('#avatar').click();
    // TODO add better identifiers to react Link component. make it an anchor tag?
    cy.get('.ant-dropdown').contains(/sign out/i).should('have.lengthOf', 1);
    cy.get('.ant-dropdown').contains(/sign out/i).click();
    cy.checkLoggedOut();
  });

  it('should be able to log in', () => {
    cy.logout();
    cy.visit(LOGIN_ROUTE);
    const username = 'determined';
    // We directly set the value to avoid using the less reliable .type() method
    // from Cypress. We also trigger 'input' event to keep it closer to an actual typing
    // behavior this would help functions relying onInput.
    // cy.get('form input#login_username')
    //   .invoke('val', username)
    //   .trigger('change')
    //   .trigger('input');

    cy.get('input#login_username')
      .type(username, { delay: 100 })
      .should('have.value', username);

    cy.get('button[type="submit"]').click();
    cy.checkLoggedIn('determined');
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
