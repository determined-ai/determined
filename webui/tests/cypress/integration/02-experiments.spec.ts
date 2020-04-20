describe('experiment List', () => {
  before(() => {
    cy.login();
    cy.visit('/ui/experiments');
  });

  describe('batch buttons', () => {
    it('should have 4 buttons', () => {
      cy.get('.batchActions button').should('have.lengthOf', 4).should('be.disabled');
    });

    it('should be disabled at start', () => {
      cy.get('.batchActions button').should('be.disabled');
    });

    describe('Open TensorBoard', () => {
      it('should be enabled when all experiments are selected', () => {
        cy.get('.batchActions input[type=checkbox]').click();
        cy.get('.batchActions input[type=checkbox]')
          .should('have.lengthOf', 1).should('be.enabled');
        cy.get('.batchActions button').contains(/tensorboard/i)
          .closest('button').should('be.enabled');
        cy.get('.batchActions input[type=checkbox]').click();
      });
    });
  });

  describe('table filter', () => {
    describe('archive toggle', () => {
      it('should show archived experiments when clicked', () => {
        cy.get('#experimentsList tr').should('have.length', 3);
        cy.get('#experimentsList .filters input[type=checkbox]').click();
        cy.get('#experimentsList tr').should('have.length', 4);
        cy.get('#experimentsList .filters input[type=checkbox]').click();
        cy.get('#experimentsList tr').should('have.length', 3);
      });

      it('should default to hiding archived experiments', () => {
        cy.get('#experimentsList .filters input[type=checkbox]').should('not.have.attr', 'checked');
        cy.get('#experimentsList tr').should('have.length', 3);
        cy.get('#experimentsList tr').should('not.contain', 'Yes');
      });
    });
  });
});
