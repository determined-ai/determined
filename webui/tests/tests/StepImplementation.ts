/* eslint-disable no-unused-vars */
import {
  AfterSpec,
  AfterSuite,
  BeforeSpec,
  BeforeSuite,
  ExecutionContext,
  Step,
  Table,
} from 'gauge-ts';
/* eslint-enable no-unused-vars */

import * as assert from 'assert';
import * as t from 'taiko';

/*
 * A require is used here to allow for plugins
 * such as video to come through.
 * Using an import will not work.
 */
const {
  $,
  button,
  clear,
  closeBrowser,
  currentURL,
  focus,
  link,
  near,
  openBrowser,
  switchTo,
  tableCell,
  text,
  waitFor,
  within,
  write,
  video,
} = require('taiko');

const HEADLESS = process.env.HEADLESS === 'true';
const HOST = process.env.DET_MASTER || 'localhost:8080';
const BASE_PATH = process.env.PUBLIC_URL || '/det';
const BASE_URL = `${HOST}${BASE_PATH}`;
const viewports = {
  desktop: { width: 1366, height: 768 },
};

/* Helper functions */

const clickAndWaitForPage = async (selector: t.SearchElement | t.MouseCoordinates) => {
  await t.click(selector, { waitForEvents: ['loadEventFired'] });
};

const goto = async (url: string) => {
  await t.goto(url, { waitForEvents: ['loadEventFired'] });
};

export default class StepImplementation {
  @BeforeSuite()
  public async beforeSuite() {
    const defaultArgs = [`--window-size=${viewports.desktop.width},${viewports.desktop.height}`];
    const browserArgs = HEADLESS
      ? { headless: true, args: [...defaultArgs, '--no-sandbox', '--disable-setuid-sandbox'] }
      : { headless: false, args: defaultArgs };
    await openBrowser(browserArgs);
  }

  @AfterSuite()
  public async afterSuite() {
    await closeBrowser();
  }

  @BeforeSpec()
  public async beforeSpec(context: ExecutionContext) {
    const spec = context.getCurrentSpec().getName();
    const filename = `reports/videos/${spec}.mp4`;
    await video.startRecording(filename);
  }

  @AfterSpec()
  public async afterSpec() {
    await video.stopRecording();
  }

  /* Authentication Steps */

  @Step('Sign in as <username> with <password>')
  public async signInWithPassword(username: string, password: string) {
    await goto(`${BASE_URL}/login`);
    await clear(focus(t.textBox({ id: 'login_username' })));
    await write(username);
    if (password !== '') {
      await focus(t.textBox({ id: 'login_password' }));
      await write(password);
    }
    await clickAndWaitForPage(button('Sign In'));
    await text(username, near($('#avatar'))).exists();
  }

  @Step('Sign in as <username> without password')
  public async signIn(username: string) {
    await this.signInWithPassword(username, '');
  }

  @Step('Sign out')
  public async signOut() {
    await t.click($('#avatar'));
    await t.click(link('Sign Out'));
    await button('Sign In').exists();
  }

  /* Browser Utility Steps */

  @Step('Should not have element <selector> present')
  public async hasNoElement(selector: string) {
    const exists = await t.$(selector).exists(undefined, 2000);
    assert.ok(!exists);
  }

  @Step('Should have element <selector> present')
  public async hasElement(selector: string) {
    await t.$(selector).exists();
  }

  @Step('Switch to mobile view')
  public async setMobileViewport() {
    await t.emulateDevice('iPhone 7');
  }

  @Step('Switch to desktop view')
  public async setDesktopViewport() {
    await t.setViewPort(viewports.desktop);
  }

  /* Navigation Steps */

  @Step('Navigate to the following routes <table>')
  public async navigateWithTable(table: Table) {
    for (var row of table.getTableRows()) {
      const label = row.getCell('label');
      await clickAndWaitForPage(link(label, within($('[class*=Navigation_base]'))));

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
        await t.closeTab();
      }
    }
  }

  @Step('Navigate to React page at <path>')
  public async navigateToReactPage(path: string) {
    await goto(`${BASE_URL}${path}`);
  }

  @Step('Navigate to dashboard page')
  public async navigateToDashboard() {
    await goto(`${BASE_URL}/dashboard`);
  }

  @Step('Navigate to experiment list page')
  public async navigateToExperimentList() {
    await goto(`${BASE_URL}/experiments`);
  }

  @Step('Navigate to experiment <id> page')
  public async navigateToExperimentDetail(id: string) {
    await goto(`${BASE_URL}/experiments/${id}`);
  }

  @Step('Navigate to task list page')
  public async navigateToTaskList() {
    await goto(`${BASE_URL}/tasks`);
  }

  @Step('Navigate to master logs page')
  public async navigateToMasterLogs() {
    await goto(`${BASE_URL}/logs`);
  }

  /* Table Steps */

  @Step('Should have <count> table rows')
  public async checkTableRowCount(count: string) {
    const expectedCount = parseInt(count);
    await assert.strictEqual((await $('tr[data-row-key]').elements()).length, expectedCount);
  }

  @Step('Sort table by column <column>')
  public async sortTableByColumn(column: string) {
    await t.click(text(column));
  }

  @Step('Toggle all table row selection')
  public async toggleAllTableRowSelection() {
    await t.click($('th input[type=checkbox]'));
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

  @Step('<action> all table rows')
  public async actionOnAllExperiments(action: string) {
    await t.click(button(action, within($('[class*=TableBatch_base]'))));
    // Wait for the modal to animate in
    await waitFor(async () => !(await $('.ant-modal.zoom-enter').exists()));
    await t.click(button(action, within($('.ant-modal-body'))));
    // Wait for the modal to animate away
    await waitFor(async () => !(await $('.ant-modal.zoom-leave').exists()));
  }

  /* Notebook and TensorBoard Steps */

  @Step('Launch notebook')
  public async launchNotebook() {
    await clickAndWaitForPage(button('Launch Notebook'));
  }

  @Step('Launch cpu-only notebook')
  public async launchCpuNotebook() {
    await t.click($('[class*=Navigation_launchIcon]'));
    await clickAndWaitForPage(text('Launch CPU-only Notebook'));
  }

  @Step('Launch tensorboard')
  public async launchTensorboard() {
    await clickAndWaitForPage(button('View in TensorBoard'));
  }

  @Step('Close wait page tab')
  public async closeTab() {
    await t.closeTab(/http.*\/(wait|proxy)/);
  }

  /* Dashboard Page Steps */

  @Step('Should have <count> recent task cards')
  public async checkRecentTasks(count: string) {
    const expectedCount = parseInt(count);
    await assert.strictEqual((await $('[class*=TaskCard_base]').elements()).length, expectedCount);
  }

  /* Experiment List Page Steps */

  @Step('<action> experiment row <row>')
  public async modifyExperiment(action: string, row: string) {
    await t.click(tableCell({ row: parseInt(row) + 1, col: 11 }));
    await t.click(text(action, within($('.ant-dropdown'))));
  }

  @Step('Toggle show archived button')
  public async toggleShowArchived() {
    await t.click(text('Show Archived'));
  }

  /* Experiment Detail Page Steps */

  @Step('Archive experiment')
  public async archiveExperiment() {
    await t.click(button('Archive'));
  }

  @Step('Unarchive experiment')
  public async unarchiveExperiment() {
    await t.click(button('Unarchive'));
  }

  @Step('Kill experiment')
  public async killExperiment() {
    await t.click(button('Kill'));
    await t.click(button('Yes'));
  }

  @Step('View experiment in TensorBoard')
  public async viewExperimentInTensorBoard() {
    await t.click(button('View in TensorBoard'));
  }

  /* Task List Page Steps */

  @Step('Filter tasks by type <table>')
  public async filterTasksByType(table: Table) {
    for (var row of table.getTableRows()) {
      const ariaLabel = row.getCell('aria-label');
      const count = row.getCell('count');
      await t.click($(`[aria-label=${ariaLabel}]`));
      await this.checkTableRowCount(count);
      await t.click($(`[aria-label=${ariaLabel}]`));
    }
  }
  /* Cluster */

  @Step('Should show <count> resource pool cards')
  public async checkResourcePoolCardCount(count: string) {
    const elements = await t.$('div[class^="ResourcePoolCard_base"]').elements();
    assert.strictEqual(elements.length, parseInt(count));
  }

  @Step('Should show <count> agents in stats')
  public async checkAgentCountStats(count: string) {
    const stats = await t.$('div[class^="OverviewStats_base"]').elements();
    assert.strictEqual(stats.length, 3);
    const numAgents = (await stats[0].text()).replace('Number of Agents', '');
    assert.strictEqual(parseInt(numAgents), parseInt(count));
  }
}
