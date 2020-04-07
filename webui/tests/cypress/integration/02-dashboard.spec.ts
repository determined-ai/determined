describe('Dashboard', () => {
  before(() => {
    cy.visit('/');
  });

  describe('Recent Tasks', () => {
    it('should show task cards', () => {
      cy.get('#recent-tasks').dataCy('task-card').should('have.length.gt', 0);
    });

    it('should show 3 cards', () => {
      cy.get('#recent-tasks').dataCy('task-card').should('have.length', 3);
    });

  });

});
