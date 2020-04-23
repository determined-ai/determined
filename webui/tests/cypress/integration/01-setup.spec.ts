describe('setup', () => {
  before(() => {
    cy.login({ username: 'determined' });
  });

  beforeEach(() => {
    cy.visit('/ui/experiments');
  });

  it('should have 4 experiments listed', () => {
    cy.get('#experimentsList tr').should('have.lengthOf', 4);
  });

  it('should have 4 active experiments listed', () => {
    cy.get('#experimentsList tr').should('have.lengthOf', 4)
      .each(($tr) => {
        cy.wrap($tr).should('contain', 'Active');
      });
  });

  it('should pause all experiments listed', () => {
    cy.get('.batchActions input[type=checkbox]').click();
    cy.get('.batchActions').contains(/pause selected/i).click();
    cy.get('.modal button').contains(/pause/i).click();
    /* eslint-disable-next-line cypress/no-unnecessary-waiting */
    cy.wait(5000);
    cy.get('#experimentsList tr')
      .each(($tr) => {
        cy.wrap($tr).should('contain', 'Paused');
      });
    cy.get('.batchActions input[type=checkbox]').click();
  });

  it('should be able to unpause experiment 1', () => {
    cy.get('#experimentsList tr td:nth-child(2)').contains('1').click();
    cy.contains('Activate').click();
    cy.visit('/ui/experiments');
    cy.get('#experimentsList tr').should('contain', 'Active');
  });

  it('should cancel experiment 2', () => {
    cy.get('#experimentsList tr td:nth-child(2)').contains('2').click();
    cy.contains('Cancel').click();
    cy.get('.modal button').contains(/cancel/i).click();
    cy.contains('Canceled', { timeout: Cypress.config('responseTimeout') });
  });

  it('should archive experiment 2', () => {
    cy.get('#experimentsList tr td:nth-child(2)').contains('2').click();
    cy.contains('Canceled');
    cy.get('body').should('not.contain', /archived/i);
    cy.contains('Archive').click();
    cy.contains(/archived/i);
    cy.visit('/ui/experiments');
    cy.get('#experimentsList tr').should('have.lengthOf', 3);
  });
});
