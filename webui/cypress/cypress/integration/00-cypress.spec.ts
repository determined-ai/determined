describe('cypress', () => {

  it('should have clean local storage', () => {
    window.localStorage.setItem('prop1', '1');
    cy.clearLocalStorage().then((ls) => {
      expect(ls.getItem('prop1')).to.be.null;
    });
  });

  describe('auth helpers', () => {
    it('should be logged out', () => {
      cy.visit('/det');
      cy.checkLoggedOut();
    });

    it('should cy.login should log in', () => {
      cy.login();
      cy.checkLoggedIn();
    });
  });

  describe('hooks', () => {
    beforeEach(() => {
      cy.login();
    });

    describe('A', () => {
      it('should be logged in', () => {
        cy.checkLoggedIn();
      });
      it('should still be logged in', () => {
        cy.checkLoggedIn();
      });
    });

    describe('B', () => {
      before(() => {
        cy.visit('/det');
      });
      it('should still be logged in', () => {
        cy.checkLoggedIn();
      });
    });
  });

});
