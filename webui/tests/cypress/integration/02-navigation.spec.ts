describe('Navigation', () => {
  before(() => {
    cy.login();
  });

  const pageTitleSelector = '[class*="Page_base_"]';
  const sectionTitleSelector = '[class*="Section_title_"]';

  describe('paths', () => {

    it('path / should display dashboard', () => {
      cy.visit('/');
      cy.get(sectionTitleSelector).contains('Recent Tasks');
      cy.get(sectionTitleSelector).contains('Overview');
    });

    it('path /det/dashboard should display dashboard', () => {
      cy.visit('/det/dashboard');
      cy.get(sectionTitleSelector).contains('Recent Tasks');
      cy.get(sectionTitleSelector).contains('Overview');
    });

    it('path /det/experiments should display experiments', () => {
      cy.visit('/det/experiments');
      cy.get(pageTitleSelector).contains('Experiments');
    });

    it('path /det/tasks should display Tasks', () => {
      cy.visit('/det/tasks');
      cy.get(pageTitleSelector).contains('Tasks');
    });

    it('path /det/cluster should display Cluster', () => {
      cy.visit('/det/cluster');
      cy.get(pageTitleSelector).contains('Cluster');
    });

    it.skip('path /det/logs should display Master Logs', () => {
      cy.visit('/det/logs');
      cy.get(pageTitleSelector).contains('Master Logs');
    });

    it.skip('path /det/trials/:id/logs should display Trial Logs', () => {
      cy.visit('/det/trials/1/logs');
      cy.get(pageTitleSelector).contains('Logs for Trial');
    });
  });

  describe('side menu buttons', () => {
    const SPAs = [ '/det', '/ui' ];

    it('clicking experiments in side menu should navigate to experiments', () => {
      SPAs.forEach(page => {
        cy.visit(page);
        cy.get('#side-menu').contains(/experiments/i).click();
        return cy.get(pageTitleSelector).contains('Experiments');
      });
    });

    it('clicking tasks in side menu should navigate to tasks', () => {
      SPAs.forEach(page => {
        cy.visit(page);
        cy.get('#side-menu').contains(/tasks/i).click();
        cy.get(pageTitleSelector).contains('Tasks');
      });
    });

    it('clicking cluster in side menu should navigate to cluster', () => {
      SPAs.forEach(page => {
        cy.visit(page);
        cy.get('#side-menu').contains(/cluster/i).click();
        cy.get(pageTitleSelector).contains('Cluster');
      });
    });

    it('clicking dashboard in side menu should navigate to dashboard', () => {
      SPAs.forEach(page => {
        cy.visit(page);
        cy.get('#side-menu').contains(/dashboard/i).click();
        cy.get(sectionTitleSelector).contains('Recent Tasks');
        cy.get(sectionTitleSelector).contains('Overview');
      });
    });
  });
});
