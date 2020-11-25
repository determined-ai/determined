import { DEFAULT_WAIT_TIME } from '../constants';

describe('Task List', () => {
  const notebookLaunchSelector =
    '[class*="Navigation_launch_"] button[class*="Navigation_launchButton_"]';
  const recordSelector = 'tr[data-row-key]';
  const batchSelector = '[class*="TableBatch_actions_"] button';
  const overflowSelector = '.ant-dropdown .ant-dropdown-menu-item';

  beforeEach(() => {
    cy.login();
    cy.visit('/det/tasks');
    cy.wait(500);
  });

  describe('Notebooks', () => {
    it('should launch notebooks', () => {
      cy.get('button[aria-label="Notebook"]').click();
      cy.server();
      cy.route('POST', /\/notebook.*/).as('createRequest');
      cy.get(notebookLaunchSelector).click();
      cy.get(notebookLaunchSelector).click();
      cy.wait('@createRequest');
      cy.visit('/det/tasks');
      cy.get(recordSelector).should('have.lengthOf', 2);
    });

    it('should terminate notebook', () => {
      cy.get('button[aria-label="Notebook"]').click();
      cy.get(`${recordSelector}:first .ant-dropdown-trigger`).click();
      cy.get(overflowSelector).contains(/kill/i).click();
      // Using the server/route approach to detect endpoint calls does not work with new API
      cy.wait(DEFAULT_WAIT_TIME);
      cy.visit('/det/tasks');
      cy.get(recordSelector).contains(/terminated/i).should('be.visible');
    });
  });

  describe('Tensorboards', () => {
    it('should launch tensorboard', () => {
      cy.get('button[aria-label="Tensorboard"]').click();
      cy.server();
      cy.route('POST', /\/tensorboard.*/).as('createRequest');
      cy.visit('/det/experiments');
      cy.get('thead input[type=checkbox]').click();
      cy.get(batchSelector)
        .contains(/view in tensorboard/i)
        .click();
      cy.wait('@createRequest');
      cy.visit('/det/tasks');
      cy.get(recordSelector).should('have.lengthOf', 1);
    });

    it('should terminate tensorboard', () => {
      cy.get('button[aria-label="Tensorboard"]').click();
      cy.get(`${recordSelector}:first .ant-dropdown-trigger`).click();
      cy.get(overflowSelector).contains(/kill/i).click();
      // Using the server/route approach to detect endpoint calls does not work with new API
      cy.wait(DEFAULT_WAIT_TIME);
      cy.visit('/det/tasks');
      cy.get(recordSelector).contains(/terminated/i).should('be.visible');
    });
  });

  describe('batch buttons', () => {
    it('should have 1 button', () => {
      cy.get('thead input[type=checkbox]').click();
      cy.get(batchSelector).should('have.lengthOf', 1);
      cy.get(batchSelector).click();
      cy.get('.ant-modal-body button').contains(/kill/i).click();
    });
  });
});
