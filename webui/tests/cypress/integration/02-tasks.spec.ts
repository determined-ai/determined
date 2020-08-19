describe('Task List', () => {
  const recordSelector = 'tr.ant-table-row';

  before(() => {
    cy.login();
    cy.visit('/det/tasks');
  });

  describe('Notebooks', () => {
    before(() => {
      cy.visit('/det/tasks');
      cy.get('button[aria-label="Notebook"]').click();
    });

    it('should launch notebooks', () => {
      cy.server();
      cy.route('POST', /\/notebook.*/).as('createRequest');
      cy.get('[class*="Navigation_launch_"] button').contains(/launch notebook/i).click().click();
      cy.wait('@createRequest');
      cy.visit('/det/tasks');
      cy.get(recordSelector).should('have.lengthOf', 2);
    });

    it('should terminate notebook', () => {
      cy.server();
      cy.route('DELETE', /\/notebook.*/).as('terminateRequest');
      cy.get(`${recordSelector}:first-child .ant-dropdown-trigger`).click();
      cy.get('.ant-dropdown .ant-dropdown-menu-item').contains(/kill/i).click();
      cy.wait('@terminateRequest');
      cy.visit('/det/tasks');
      cy.get(recordSelector).contains(/terminated/i).should('be.visible');
    });

    after(() => {
      cy.get('button[aria-label="Notebook"]').click();
    });
  });

  describe('Tensorboards', () => {
    before(() => {
      cy.visit('/det/tasks');
      cy.get('button[aria-label="Tensorboard"]').click();
    });

    it('should launch tensorboard', () => {
      cy.server();
      cy.route('POST', /\/tensorboard.*/).as('createRequest');
      cy.visit('/det/experiments');
      cy.get('thead input[type=checkbox]').click();
      cy.get('[class*="TableBatch_actions_"] button')
        .contains(/view in tensorBoard/i)
        .click();
      cy.wait('@createRequest');
      cy.visit('/det/tasks');
      cy.get(recordSelector).should('have.lengthOf', 1);
    });

    it('should terminate tensorboard', () => {
      cy.server();
      cy.route('DELETE', /\/tensorboard.*/).as('terminateRequest');
      cy.get(`${recordSelector}:first-child .ant-dropdown-trigger`).click();
      cy.get('.ant-dropdown .ant-dropdown-menu-item').contains(/kill/i).click();
      cy.wait('@terminateRequest');
      cy.visit('/det/tasks');
      cy.get(recordSelector).contains(/terminated/i).should('be.visible');
    });

    after(() => {
      cy.get('button[aria-label="Tensorboard"]').click();
    });
  });

  describe('batch buttons', () => {
    it('should have 1 button', () => {
      cy.get('thead input[type=checkbox]').click();
      cy.get('[class*="TableBatch_actions_"] button').should('have.lengthOf', 1);
    });
  });
});
