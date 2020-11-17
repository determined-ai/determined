describe('Dashboard', () => {

  beforeEach(() => {
    cy.visit('/');
  });

  describe('Recent Tasks', () => {
    it('should show task cards', () => {
      cy.get('#recent-tasks [class^="TaskCard_base"]').should('have.length.gt', 0);
    });

    it('should show 3 cards', () => {
      cy.get('#recent-tasks [class^="TaskCard_base"]').should('have.length', 3);
    });
  });
});
