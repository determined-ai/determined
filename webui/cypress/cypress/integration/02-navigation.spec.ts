describe('Navigation', () => {
  const pageTitleSelector = '[class*="Page_base_"]';
  const sectionTitleSelector = '[class*="Section_title_"]';
  const navSelector = '[class*="Navigation_base_"]';

  describe('paths', () => {
    beforeEach(() => {
      cy.login();
    });

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

    it('path /det/logs should display Master Logs', () => {
      cy.visit('/det/logs');
      cy.get(pageTitleSelector).contains('Master Logs');
    });

    it('path /det/trials/:id/logs should display Trial Logs', () => {
      cy.visit('/det/trials/1/logs');
      cy.get(pageTitleSelector).contains('Trial 1 Logs');
    });
  });

  describe('side menu buttons', () => {
    beforeEach(() => {
      cy.login();
      cy.visit('/det');
    });

    it('clicking experiments on navigation should navigate to experiments', () => {
      cy.get(navSelector).contains(/experiments/i).click();
      return cy.get(pageTitleSelector).contains('Experiments');
    });

    it('clicking tasks on navigation should navigate to tasks', () => {
      cy.get(navSelector).contains(/tasks/i).click();
      cy.get(pageTitleSelector).contains('Tasks');
    });

    it('clicking cluster on navigation should navigate to cluster', () => {
      cy.get(navSelector).contains(/cluster/i).click();
      cy.get(pageTitleSelector).contains('Cluster');
    });

    it('clicking dashboard on navigation should navigate to dashboard', () => {
      cy.get(navSelector).contains(/dashboard/i).click();
      cy.get(sectionTitleSelector).contains('Recent Tasks');
      cy.get(sectionTitleSelector).contains('Overview');
    });
  });
});
