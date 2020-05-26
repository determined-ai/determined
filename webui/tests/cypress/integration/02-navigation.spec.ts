describe('Navigation', () => {
  before(() => {
    cy.login();
  });

  const titleSelector = '#det-main-container div.text-2xl';

  describe('paths', () => {

    it('path / should display dashboard', () => {
      cy.visit('/');
      cy.get('section h5').contains('Recent Tasks');
      cy.get('section h5').contains('Overview');
    });

    it('path /det/dashboard should display dashboard', () => {
      cy.visit('/det/dashboard');
      cy.get('section h5').contains('Recent Tasks');
      cy.get('section h5').contains('Overview');
    });

    it('path /ui/experiments should display experiments', () => {
      cy.visit('/ui/experiments');
      cy.get(titleSelector).contains('Experiments');
    });

    it('path /ui/notebooks should display Notebooks', () => {
      cy.visit('/ui/notebooks');
      cy.get(titleSelector).contains('Notebooks');
    });

    it('path /ui/tensorboards should display TensorBorads', () => {
      cy.visit('/ui/tensorboards');
      cy.get(titleSelector).contains('TensorBoards');
    });

    it('path /ui/cluster should display cluster', () => {
      cy.visit('/ui/cluster');
      cy.get(titleSelector).contains('Cluster');
    });

    it('path /ui/shells should display Shells', () => {
      cy.visit('/ui/shells');
      cy.get(titleSelector).contains('Shells');
    });

    it('path /ui/commands should display Commands', () => {
      cy.visit('/ui/commands');
      cy.get(titleSelector).contains('Commands');
    });
  });

  describe('side menu buttons', () => {
    it('clicking experiments in side menu should navigate to experiments', () => {
      cy.visit('/det/dashboard');
      cy.get('#side-menu').contains(/experiments/i).click();
      cy.get(titleSelector).contains('Experiments');
    });

    it('clicking notebooks in side menu should navigate to notebooks', () => {
      cy.visit('/ui/experiments');
      cy.get('#side-menu').contains(/notebooks/i).click();
      cy.get(titleSelector).contains('Notebooks');
    });

    it('clicking tensorboards in side menu should navigate to tensorboards', () => {
      cy.visit('/ui/notebooks');
      cy.get('#side-menu').contains(/tensorboards/i).click();
      cy.get(titleSelector).contains('TensorBoards');
    });

    it('clicking cluster in side menu should navigate to cluster', () => {
      cy.visit('/ui/tensorboards');
      cy.get('#side-menu').contains(/cluster/i).click();
      cy.get(titleSelector).contains('Cluster');
    });

    it('clicking shells in side menu should navigate to shells', () => {
      cy.visit('/ui/cluster');
      cy.get('#side-menu').contains(/shells/i).click();
      cy.get(titleSelector).contains('Shells');
    });

    it('clicking commands in side menu should navigate to commands', () => {
      cy.visit('/ui/shells');
      cy.get('#side-menu').contains(/commands/i).click();
      cy.get(titleSelector).contains('Commands');
    });

    it('clicking dashboard in side menu should navigate to dashboard', () => {
      cy.visit('/ui/commands');
      cy.get('#side-menu').contains(/dashboard/i).click();
      cy.get('section h5').contains('Recent Tasks');
      cy.get('section h5').contains('Overview');
    });
  });
});
