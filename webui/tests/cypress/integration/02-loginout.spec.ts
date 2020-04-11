describe('Sign in/out', () => {

  function checkLoggedIn(username: string): void {
    // Check for the presence/absence of the icons for the user dropdown and
    // cluster page link in the top bar, which should be present if and only if
    // the user is logged in.
    cy.get('#avatar').should('exist');
    cy.get('#avatar').should('have.text', username.charAt(0).toUpperCase());
  }

  function checkLoggedOut(): void {
    cy.get('#avatar').should('not.exist');
  }

  function ensureLoggedOut(): void {
    cy.visit('/ui/logout');
    checkLoggedOut();
  }

  it('should log in', () => {
    cy.visit('/ui/experiments');
    checkLoggedIn('determined');
  });

  it('should log out', () => {
    // Open the dropdown menu and click the button to log out.
    cy.get('#avatar').click();
    cy.get('nav a[href="/ui/logout"]').should('have.lengthOf', 1);
    cy.get('nav a[href="/ui/logout"]').click();
    checkLoggedOut();
  });

  it('should log back in after logging out', () => {
    // Logging out above should put us on the login page, so enter the login
    // information directly.
    ensureLoggedOut();
    cy.visit('/ui/login');
    const username = 'determined';
    // We directly set the value to avoid using the less reliable .type() method
    // from Cypress. We also trigger 'input' event to keep it closer to an actual typing
    // behavior this would help functions relying onInput.
    // cy.get('input#input-username')
    //   .invoke('val', username)
    //   .trigger('input');

    cy.get('input#input-username')
      .type(username, { delay: 50, force: true });

    cy.get('input#input-username')
      .should('have.value', username);

    cy.get('button[type="submit"]').click();
    checkLoggedIn('determined');
  });
});
