describe('Navigation', () => {
  before(() => {
    cy.login();
  });

  const elmTitleSelector = '#det-main-container div.text-2xl';
  const pageTitleSelector = '[class^="Page_title_"]';
  const sectionTitleSelector = '[class^="Section_title_"]';

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
      cy.get(elmTitleSelector).contains('Experiments');
    });

    it('path /ui/notebooks should display Notebooks', () => {
      cy.visit('/ui/notebooks');
      cy.get(elmTitleSelector).contains('Notebooks');
    });

    it('path /ui/tensorboards should display TensorBorads', () => {
      cy.visit('/ui/tensorboards');
      cy.get(elmTitleSelector).contains('TensorBoards');
    });

    it('path /det/cluster should display Cluster', () => {
      cy.visit('/det/cluster');
      cy.get(pageTitleSelector).contains('Cluster');
    });

    it('path /ui/shells should display Shells', () => {
      cy.visit('/ui/shells');
      cy.get(elmTitleSelector).contains('Shells');
    });

    it('path /ui/commands should display Commands', () => {
      cy.visit('/ui/commands');
      cy.get(elmTitleSelector).contains('Commands');
    });

    it.skip('path /det/logs should display Master Logs', () => {
      cy.visit('/det/logs');
      cy.get(sectionTitleSelector).contains('Master Logs');
    });

    it.skip('path /det/trials/:id/logs should display Trial Logs', () => {
      cy.visit('/det/trials/1/logs');
      cy.get(sectionTitleSelector).contains('Logs for Trial');
    });
  });

  describe('side menu buttons', () => {
    const SPAs = [ '/det', '/ui' ];

    it('clicking experiments in side menu should navigate to experiments', () => {
      SPAs.forEach(page => {
        cy.visit(page);
        cy.get('#side-menu').contains(/experiments/i).click();
        return cy.get(elmTitleSelector).contains('Experiments');
      });
    });

    it('clicking notebooks in side menu should navigate to notebooks', () => {
      SPAs.forEach(page => {
        cy.visit(page);
        cy.get('#side-menu').contains(/notebooks/i).click();
        cy.get(elmTitleSelector).contains('Notebooks');
      });
    });

    it('clicking tensorboards in side menu should navigate to tensorboards', () => {
      SPAs.forEach(page => {
        cy.visit(page);
        cy.get('#side-menu').contains(/tensorboards/i).click();
        cy.get(elmTitleSelector).contains('TensorBoards');
      });
    });

    it('clicking cluster in side menu should navigate to cluster', () => {
      SPAs.forEach(page => {
        cy.visit(page);
        cy.get('#side-menu').contains(/cluster/i).click();
        cy.get(pageTitleSelector).contains('Cluster');
      });
    });

    it('clicking shells in side menu should navigate to shells', () => {
      SPAs.forEach(page => {
        cy.visit(page);
        cy.get('#side-menu').contains(/shells/i).click();
        cy.get(elmTitleSelector).contains('Shells');
      });
    });

    it('clicking commands in side menu should navigate to commands', () => {
      SPAs.forEach(page => {
        cy.visit(page);
        cy.get('#side-menu').contains(/commands/i).click();
        cy.get(elmTitleSelector).contains('Commands');
      });
    });

    it('clicking dashboard in side menu should navigate to dashboard', () => {
      SPAs.forEach(page => {
        cy.visit(page);
        cy.get('#side-menu').contains(/dashboard/i).click();
        cy.get('section h5').contains('Recent Tasks');
        cy.get('section h5').contains('Overview');
      });
    });
  });
});
