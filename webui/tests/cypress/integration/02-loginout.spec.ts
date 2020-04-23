describe('Sign in/out', () => {

  function ensureLoggedOut(): void {
    cy.visit('/det/logout');
    cy.checkLoggedOut();
  }

  it('should log in', () => {
    cy.visit('/det/experiments');
    cy.checkLoggedIn('determined');
  });

  it('should log out', () => {
    // Open the dropdown menu and click the button to log out.
    cy.get('#avatar').click();
    cy.get('nav a[href="/det/logout"]').should('have.lengthOf', 1);
    cy.get('nav a[href="/det/logout"]').click();
    cy.checkLoggedOut();
  });

  it('should log back in after logging out', () => {
    // Logging out above should put us on the login page, so enter the login
    // information directly.
    ensureLoggedOut();
    cy.visit('/det/login');
    const username = 'determined';
    // We directly set the value to avoid using the less reliable .type() method
    // from Cypress. We also trigger 'input' event to keep it closer to an actual typing
    // behavior this would help functions relying onInput.
    // cy.get('input#input-username')
    //   .invoke('val', username)
    //   .trigger('input');

    cy.get('input#basic-username')
      .type(username, { delay: 50, force: true })
      .should('have.value', username);

    cy.get('button[type="submit"]').click();
    cy.checkLoggedIn('determined');
  });
});
