describe('Experiment List', () => {
  const recordSelector = 'tr.ant-table-row';

  beforeEach(() => {
    cy.visit('/det/experiments');
  });

  describe('batch action buttons', () => {
    it('should have 7 buttons', () => {
      cy.get('thead input[type=checkbox]').click();
      cy.get('[class*="TableBatch_actions_"] button').should('have.lengthOf', 7);
    });

    it('should have 2 disabled buttons', () => {
      cy.get('thead input[type=checkbox]').click();
      cy.get('[class*="TableBatch_actions_"] button[disabled]').should('have.lengthOf', 2);
    });

    describe('View in TensorBoard', () => {
      it('should be enabled when all experiments are selected', () => {
        cy.get('thead input[type=checkbox]').click();
        cy.get('[class*="TableBatch_actions_"] button:first-child')
          .should('contain', 'View in TensorBoard')
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

      it('should show archived column', () => {
        cy.get('[class*="Toggle_base_"] button').click();
        cy.get(`${recordSelector} .icon-checkmark`).should('have.lengthOf', 1);
      });
    });
  });
});
