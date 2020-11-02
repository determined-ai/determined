describe('Experiment List', () => {
  const recordSelector = '.ant-table-tbody tr.ant-table-row';
  const batchSelector = '[class*="TableBatch_actions_"] button';
  const toggleSelector = '[class*="Toggle_base_"] button';

  beforeEach(() => {
    cy.login();
    cy.visit('/det/experiments');
    cy.wait(5000);
  });

  describe('batch action buttons', () => {
    it('should have 7 buttons', () => {
      cy.get('thead input[type=checkbox]').click();
      cy.get(batchSelector).should('have.lengthOf', 7);
    });

    it('should have 2 disabled buttons', () => {
      cy.get('thead input[type=checkbox]').click();
      cy.get(`${batchSelector}[disabled]`).should('have.lengthOf', 2);
    });

    describe('View in TensorBoard', () => {
      it('should be enabled when all experiments are selected', () => {
        cy.get('thead input[type=checkbox]').click();
        cy.get(`${batchSelector}:first`)
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
        cy.get(toggleSelector).click();
        cy.get(recordSelector).should('have.length', 4);
        cy.get(toggleSelector).click();
        cy.get(recordSelector).should('have.length', 3);
      });

      it('should show archived column', () => {
        cy.get(toggleSelector).click();
        cy.get(`${recordSelector} .icon-checkmark`).should('have.lengthOf', 1);
      });
    });
  });
});
