/* eslint-disable no-unused-vars */
import {
  AfterScenario,
  AfterSpec,
  AfterSuite,
  BeforeScenario,
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
  click,
  closeBrowser,
  closeTab,
  currentURL,
  focus,
  goto,
  link,
  near,
  openBrowser,
  switchTo,
  tableCell,
  text,
  textBox,
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
    await click(text(column));
  }

  @Step('Select all table rows')
  public async selectAllTableRows() {
    await click($('th input[type=checkbox]'));
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
    await click(button(action, within($('[class*=TableBatch_base]'))));
    // Wait for the modal to animate in
    await waitFor(async () => !(await $('.ant-modal.zoom-enter').exists()));
    await click(button(action, within($('.ant-modal-body'))));
    // Wait for the modal to animate away
    await waitFor(async () => !(await $('.ant-modal.zoom-leave').exists()));
  }

  /* Notebook and TensorBoard Steps */

  @Step('Launch notebook')
  public async launchNotebook() {
    await click(button('Launch Notebook'), { waitForEvents: ['targetNavigated'] });
    await /http(.*)(wait|proxy)/;
  }

  @Step('Launch cpu-only notebook')
  public async launchCpuNotebook() {
    await click($('[class*=Navigation_launchIcon]'));
    await click(text('Launch CPU-only Notebook'));
  }

  @Step('Launch tensorboard')
  public async launchTensorboard() {
    await click(button('View in TensorBoard'), { waitForEvents: ['targetNavigated'] });
  }

  @Step('Close current tab')
  public async closeWaitPage() {
    await closeTab();
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
    await click(tableCell({ row: parseInt(row) + 1, col: 11 }));
    await click(text(action, within($('.ant-dropdown'))));
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
