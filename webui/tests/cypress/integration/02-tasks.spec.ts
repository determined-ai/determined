describe('Task List', () => {
  const recordSelector = 'tr.ant-table-row';

  before(() => {
    cy.login();
    cy.visit('/det/tasks');
  });

  describe('launch notebooks', () => {
    it('should launch notebooks', () => {
      cy.visit('/ui/notebooks');
      cy.get('button').contains(/launch new notebook/i).click().click();
      cy.visit('/det/tasks');
      cy.get(recordSelector).should('have.length', 2);
    });
  });

  describe('launch tensorboards', () => {
    it('should launch tensorboards', () => {
      cy.visit('/det/experiments');
      cy.get('thead input[type=checkbox]').click();
      cy.get('[class*="TableBatch_actions_"] button:first-child')
        .should('contain', 'Open TensorBoard')
        .click();
      cy.visit('/det/tasks');
      cy.get(recordSelector).should('have.length', 3);
    });
  });

  describe('batch buttons', () => {
    it('should have 1 button', () => {
      cy.get('thead input[type=checkbox]').click();
      cy.get('[class*="TableBatch_actions_"] button').should('have.lengthOf', 1);
    });
  });

  describe('table filter', () => {
    it('should filter notebooks by task type', () => {
      cy.get('button[aria-label="Tensorboard"]').click();
      cy.get(recordSelector).should('have.length', 1);
      cy.get('button[aria-label="Tensorboard"]').click();
      cy.get(recordSelector).should('have.length', 3);
    });
  });
});
