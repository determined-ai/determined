describe('Notebooks List', () => {
  before(() => {
    cy.visit('/ui/notebooks');
  });

  describe('launching Notebooks', () => {
    it('should say no notebooks have started yet', () => {
      cy.get('.table .message').contains(/no notebooks have been started/i);
    });

    describe('launch button', () => {
      it('should start a 1 slot notebook', () => {
        cy.contains(/new notebook/i).click();
        cy.get('.table tr').should('have.lengthOf', 1);
      });

      it('should present 0 slot launch button when clicking the dropdown', () => {
        cy.get('button .fa-caret-down').click();
        cy.contains(/cpu-only/i);
        cy.get('button .fa-caret-down').click();
      });

      it('should launch a Notebook when clicking the cpu-only button', () => {
        cy.get('button .fa-caret-down').click();
        cy.contains(/launch new cpu/i).click();
        cy.get('.table tr').should('have.lengthOf', 2);
      });
    });

    describe('action buttons', () => {
      it('there should be 3 action buttons', () => {
        cy.get('tr:first-child td:last-child button').should('have.lengthOf', 3);
      });

      it.skip('should open the logs modal when clicking the logs button', () => {
        cy.get('tr:first-child').contains(/logs/i).click();
        cy.get('.modal').contains(/logs for notebook/i);
        cy.get('.modal .fa-times').click();
      });

      it('logs button should be enabled', () => {
        cy.get('button').contains(/logs/i).closest('button').should('be.enabled');
      });

      it('open button should be enabled', () => {
        cy.get('button').contains(/open/i).closest('button').should('be.enabled');
      });

      it('kill button should be enabled', () => {
        cy.get('button').contains(/kill/i, { timeout: 30000 })
          .closest('button').should('be.enabled');
      });

      it('kill should terminate the notebook', () => {
      // Look for either terminated or terminating to avoid long delays.
        cy.get('.table').should('not.contain', /terminat/i);
        cy.get('button').contains(/kill/i).click();
        cy.get('.modal button').contains(/confirm/i).click();
        cy.contains(/terminat/i);
      });

    });

  });

});
