/* eslint-disable no-unused-vars */
import {
  AfterScenario,
  AfterSuite,
  BeforeScenario,
  BeforeSuite,
  ExecutionContext,
  Step,
  Table,
} from 'gauge-ts';
/* eslint-enable no-unused-vars */

import * as assert from 'assert';

/*
 * A require is used here to allow for plugins
 * such as screencast to come through.
 * Using an import will not work.
 */
const {
  $,
  button,
  clear,
  click,
  closeBrowser,
  closeTab,
  currentURL,
  focus,
  goto,
  link,
  near,
  openBrowser,
  screencast,
  switchTo,
  tableCell,
  text,
  textBox,
  waitFor,
  within,
  write,
} = require('taiko');

const HEADLESS = process.env.HEADLESS === 'true';
const HOST = process.env.DET_MASTER || 'localhost:8080';
const BASE_PATH = process.env.PUBLIC_URL || '/det';
const BASE_URL = `${HOST}${BASE_PATH}`;
const SCREENCAST = process.env.SCREENCAST === 'true';

export default class StepImplementation {
  @BeforeSuite()
  public async beforeSuite() {
    const browserArgs = HEADLESS
      ? { headless: true, args: ['--no-sandbox', '--disable-setuid-sandbox'] }
      : { headless: false };
    await openBrowser(browserArgs);
  }

  @AfterSuite()
  public async afterSuite() {
    await closeBrowser();
  }

  @BeforeScenario()
  public async startScreencast(context: ExecutionContext) {
    if (SCREENCAST) {
      const spec = context.getCurrentSpec().getName();
      const scenario = context.getCurrentScenario().getName();
      const filename = `screencasts/${spec} -- ${scenario}.gif`;
      await screencast.startScreencast(filename);
    }
  }

  @AfterScenario()
  public async stopScreencast() {
    if (SCREENCAST) {
      await screencast.stopScreencast();
    }
  }

  /* Authentication Steps */

  @Step('Sign in as <username> with <password>')
  public async signInWithPassword(username: string, password: string) {
    await goto(`${BASE_URL}/login`);
    await clear(focus(textBox('username')));
    await write(username);
    if (password !== '') {
      await focus(textBox('password'));
      await write(password);
    }
    await click(button('Sign In'));
    await text(username, near($('#avatar'))).exists();
  }

  @Step('Sign in as <username> without password')
  public async signIn(username: string) {
    await this.signInWithPassword(username, '');
  }

  @Step('Sign out')
  public async signOut() {
    await click($('#avatar'));
    await click(link('Sign Out'));
    await button('Sign In').exists();
  }

  /* Navigation Steps */

  @Step('Navigate to the following routes <table>')
  public async navigateWithTable(table: Table) {
    for (var row of table.getTableRows()) {
      const label = row.getCell('label');
      await click(link(label, within($('[class*=Navigation_base]'))));

      const external = row.getCell('external') === 'true';
      if (external) {
        const title = row.getCell('title');
        const titleRegex = new RegExp(title, 'i');
        await switchTo(titleRegex);
      }

      const path = row.getCell('route');
      const url = await currentURL();
      assert.ok(url.includes(path));

      if (external) {
        await closeTab();
      }
    }
  }

  @Step('Navigate to sign in page')
  public async navigateToSignIn() {
    await goto(`${BASE_URL}/login`);
  }

  @Step('Navigate to dashboard page')
  public async navigateToDashboard() {
    await goto(`${BASE_URL}/dashboard`);
  }

  @Step('Navigate to experiment list page')
  public async navigateToExperimentList() {
    await goto(`${BASE_URL}/experiments`);
  }

  @Step('Navigate to task list page')
  public async navigateToTaskList() {
    await goto(`${BASE_URL}/tasks`);
  }

  /* Table Steps */

  @Step('Should have <count> table rows')
  public async checkTableRowCount(count: string) {
    const expectedCount = parseInt(count);
    await assert.strictEqual((await $('tr[data-row-key]').elements()).length, expectedCount);
  }

  @Step('Sort table by column <column>')
  public async sortTableByColumn(column: string) {
    await click(text(column));
  }

  @Step('Table batch should have following buttons <table>')
  public async checkTableBatchButton(table: Table) {
    for (var row of table.getTableRows()) {
      const label = row.getCell('table batch buttons');
      const disabled = row.getCell('disabled') === 'true';
      const batchButton = await button(label, within($('[class*=TableBatch_base]')));
      await batchButton.exists();
      assert.strictEqual(await batchButton.isDisabled(), disabled);
    }
  }

  /* Notebook and TensorBoard Steps */

  @Step('Launch notebook')
  public async launchNotebook() {
    await click(button('Launch Notebook'));
  }

  @Step('Launch cpu-only notebook')
  public async launchCpuNotebook() {
    await click($('[class*=Navigation_launchIcon]'));
    await click(text('Launch CPU-only Notebook'));
  }

  @Step('Close wait page')
  public async closeWaitPage() {
    await switchTo(/http(.*)(wait|proxy)/);
    await closeTab();
    await switchTo(/Determined/);
  }

  /* Dashboard Page Steps */

  @Step('Should have <count> recent task cards')
  public async checkRecentTasks(count: string) {
    const expectedCount = parseInt(count);
    await assert.strictEqual((await $('[class*=TaskCard_base]').elements()).length, expectedCount);
  }

  /* Experiment List Page Steps */

  @Step('Pause all experiments')
  public async pauseAllExperiments() {
    await click(button('Pause', within($('[class*=TableBatch_base]'))));
    // Wait for the modal to complete animation
    await waitFor(1000);
    await click(button('Pause', within($('.ant-modal-body'))));
    // Wait for the table batch to animate away
    await waitFor(1000);
  }

  @Step('<action> experiment row <row>')
  public async modifyExperiment(action: string, row: string) {
    await click(tableCell({ row: parseInt(row) + 1, col: 11 }));
    await click(text(action, within($('.ant-dropdown'))));
  }

  @Step('Select all table rows')
  public async selectAllTableRows() {
    await click($('th input[type=checkbox]'));
  }

  @Step('Toggle show archived button')
  public async toggleShowArchived() {
    await click(button({ class: 'ant-switch' }));
  }

  /* Task List Page Steps */

  @Step('Filter tasks by type <table>')
  public async filterTasksByType(table: Table) {
    for (var row of table.getTableRows()) {
      const ariaLabel = row.getCell('aria-label');
      const count = row.getCell('count');
      await click($(`[aria-label=${ariaLabel}]`));
      await this.checkTableRowCount(count);
      await click($(`[aria-label=${ariaLabel}]`));
    }
  }
}
