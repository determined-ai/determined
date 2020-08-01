describe('Experiment List', () => {
  const recordSelector = 'tr.ant-table-row';

  before(() => {
    cy.login();
    cy.visit('/det/experiments');
  });

  describe('batch buttons', () => {
    it('should have 7 buttons', () => {
      cy.get('thead input[type=checkbox]').click();
      cy.get('[class*="TableBatch_actions_"] button').should('have.lengthOf', 7);
    });

    describe('Open TensorBoard', () => {
      it('should be enabled when all experiments are selected', () => {
        cy.get('thead input[type=checkbox]').click();
        cy.get('[class*="TableBatch_actions_"] button:first-child')
          .should('contain', 'Open TensorBoard')
          .should('be.enabled');
        cy.get('thead input[type=checkbox]').click();
      });
    });
  });

  describe('table filter', () => {
    describe('archive toggle', () => {
      it('should show and hide archived experiments', () => {
        cy.get(recordSelector).should('have.length', 3);
        cy.get('[class*="Toggle_base_"] button').click();
        cy.get(recordSelector).should('have.length', 4);
        cy.get('[class*="Toggle_base_"] button').click();
        cy.get(recordSelector).should('have.length', 3);
      });

      // TODO: fix when Archived column has landed
      // it('should default to hiding archived experiments', () => {
      //   cy.get('#experimentsList .filters input[type=checkbox]').should('not.have.attr', 'checked');
      //   cy.get(recordSelector).should('have.length', 3);
      //   cy.get(recordSelector).should('not.contain', 'Yes');
      // });
    });
  });
});
