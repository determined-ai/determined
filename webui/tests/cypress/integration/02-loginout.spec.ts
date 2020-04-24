describe('Sign in/out', () => {

  before(() => {
    cy.login();
  });

  const elmTitleSelector = '#det-main-container div.text-2xl';

  it('should be logged in React side', () => {
    cy.visit('/det/dashboard');
    cy.checkLoggedIn('determined');
  });

  it('should be logged in Elm side', () => {
    cy.visit('/ui/experiments');
    cy.checkLoggedIn('determined');
  });

  it('should be able to log out from Elm', () => {
    // Open the dropdown menu and click the button to log out.
    cy.login();
    cy.visit('/ui');
    cy.get('#avatar').click();
    cy.get('nav a[href="/det/logout"]').should('have.lengthOf', 1);
    cy.get('nav a[href="/det/logout"]').click();
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
    cy.visit('/det/login');
    const username = 'determined';
    // We directly set the value to avoid using the less reliable .type() method
    // from Cypress. We also trigger 'input' event to keep it closer to an actual typing
    // behavior this would help functions relying onInput.
    // cy.get('form input#basic_username')
    //   .invoke('val', username)
    //   .trigger('change')
    //   .trigger('input');

    cy.get('input#basic_username')
      .type(username, { delay: 100 })
      .should('have.value', username);

    cy.get('button[type="submit"]').click();
    cy.checkLoggedIn('determined');
  });

  it('visiting login while logged in should redirect away from login', () => {
    cy.login();
    cy.visit('/det/login');
    cy.url().should('not.contain', '/det/login');
  });

  it('visting the logout page should logout the user', () => {
    cy.login();
    cy.visit('/det/logout');
    cy.checkLoggedOut();
  });

  it('visting the logout page should land user in the login page', () => {
    cy.visit('/det/logout');
    cy.url().should('contain', '/det/login');
  });

  it('login page should redirect to the request url on elm', () => {
    cy.login();
    cy.visit('/det/login?redirect=/ui/experiments');
    cy.get(elmTitleSelector).contains('Experiments');
  });

});
