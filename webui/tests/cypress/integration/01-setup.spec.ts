import { DEFAULT_WAIT_TIME, STORAGE_KEY_AUTH } from '../constants';

describe('setup', () => {
  const recordSelector = 'tr.ant-table-row';

  before(() => {
    cy.login();
    cy.saveLocalStorageCache([ STORAGE_KEY_AUTH ]);
    cy.visit('/det');
  });

  beforeEach(() => {
    cy.restoreLocalStorageCache([ STORAGE_KEY_AUTH ]);
    cy.visit('/det/experiments');
  });

  it('should have 4 experiments listed', () => {
    cy.get(recordSelector).should('have.lengthOf', 4);
  });

  it('should have 4 active or completed experiments listed', () => {
    cy.get(recordSelector)
      .should('have.lengthOf', 4)
      .each(($tr) => cy.wrap($tr).contains(/(active|completed)/i));
  });

  it('should pause all experiments listed', () => {
    cy.get('thead input[type=checkbox]').click();
    cy.get('[class*="TableBatch_base_').contains(/pause/i).click();
    cy.get('.ant-modal-body button').contains(/pause/i).click();
    /* eslint-disable-next-line cypress/no-unnecessary-waiting */
    cy.wait(DEFAULT_WAIT_TIME);
    cy.get(recordSelector)
      .each(($tr) => cy.wrap($tr).should('contain', 'Paused'));
    cy.get('thead input[type=checkbox]').click();
  });

  it('should be able to unpause experiment 1', () => {
    cy.get(`${recordSelector} td:nth-child(2)`).contains('1').click();
    cy.contains('Activate').click();
    cy.visit('/det/experiments');
    cy.get(recordSelector).should('contain', 'Active');
  });

  it('should kill experiment 2', () => {
    cy.get(`${recordSelector} td:nth-child(2)`).contains('2').click();
    cy.contains('Kill').click();
    cy.get('.ant-popover button').contains(/yes/i).click();
    cy.contains('Canceled', { timeout: Cypress.config('responseTimeout') });
  });

  it('should archive experiment 2', () => {
    cy.get(`${recordSelector} td:nth-child(2)`).contains('2').click();
    cy.contains('Canceled');
    cy.get('body').should('not.contain', /archived/i);
    cy.contains('Archive').click();
    cy.reload(); // polling is stopped on terminated experiments.
    cy.contains(/unarchive/i);
    cy.visit('/det/experiments');
    cy.get(recordSelector).should('have.lengthOf', 3);
  });
});
