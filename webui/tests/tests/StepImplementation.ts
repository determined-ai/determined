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
import * as expect from 'expect';
import * as t from 'taiko';

/*
 * A require is used here to allow for plugins such as video to come through.
 * Using an import will not work.
 */
const { video } = require('taiko');

const HEADLESS = process.env.HEADLESS === 'true';
const HOST = process.env.DET_MASTER || 'localhost:8080';
const BASE_PATH = process.env.PUBLIC_URL || '/det';
const BASE_URL = `${HOST}${BASE_PATH}`;
const viewports = {
  desktop: { width: 1366, height: 768 },
};

/* Helper functions */

const clickAndWaitForPage = async (
  selector: t.SearchElement | t.MouseCoordinates,
  external = false,
) => {
  const waitForEvents: t.BrowserEvent[] = [external ? 'targetNavigated' : 'loadEventFired'];
  await t.click(selector, { waitForEvents });
};

const clickAndCloseTab = async (selector: t.SearchElement | t.MouseCoordinates) => {
  await t.click(selector, { waitForEvents: ['targetNavigated'] });
  await t.closeTab();
};

const checkTextContentFor = async (keywords: string[], shouldExist: boolean, timeout = 1500) => {
  const promises = keywords.map(async (text) => await t.text(text).exists(undefined, timeout));
  const misses = [];
  const results = await Promise.all(promises);
  results.forEach((exists, idx) => {
    if (exists != shouldExist) {
      misses.push(keywords[idx]);
    }
  });
  expect(misses).toHaveLength(0);
};

const goto = async (url: string) => {
  await t.goto(url, { waitForEvents: ['loadEventFired'] });
};

/*
 * Taiko `elements()` often return duplicate elements after upgrading to 1.2.3.
 * Use the text values of the element as a hash to dedupe the elements.
 */
const getElements = async (
  selector: string,
  options?: t.DollarOptions | t.RelativeSearchElement,
  ...args: t.RelativeSearchElement[]
): Promise<t.Element[]> => {
  const map: Record<string, boolean> = {};
  const elements = await t.$(selector, options, ...args).elements();
  const dedupedElements = [];

  for (const element of elements) {
    const hash = (await element.text()).replace(/\s+/g, ' ').replace(/\r?\n|\r/g, '');
    console.log('HASH:', hash);
    if (!map[hash]) {
      map[hash] = true;
      dedupedElements.push(element);
    }
  }

  return dedupedElements;
};

export default class StepImplementation {
  @BeforeSuite()
  public async beforeSuite() {
    const defaultArgs = [
      `--window-size=${viewports.desktop.width},${viewports.desktop.height}`,
      '--disable-gpu',
    ];
    const browserArgs = HEADLESS
      ? { headless: true, args: [...defaultArgs, '--no-sandbox', '--disable-setuid-sandbox'] }
      : { headless: false, args: defaultArgs };
    await t.openBrowser(browserArgs);
  }

  @AfterSuite()
  public async afterSuite() {
    await t.closeBrowser();
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

  @BeforeSpec({ tags: ['skip'] })
  public async beforeSkipSpec(context: ExecutionContext) {
    const spec = context.getCurrentSpec().getName();
    throw new Error(`Skipping spec ${spec}`);
  }

  /* Authentication Steps */

  @Step('Sign in')
  public async justSignIn() {
    await this.signIn('determined');
  }

  @Step('Sign in as <username> with <password>')
  public async signInWithPassword(username: string, password: string) {
    await goto(`${BASE_URL}/login`);
    await t.clear(t.focus(t.textBox({ id: 'login_username' })));
    await t.write(username);
    if (password !== '') {
      await t.focus(t.textBox({ id: 'login_password' }));
      await t.write(password);
    }
    await clickAndWaitForPage(t.button('Sign In'));
    await t.text(username, t.near(t.$('#avatar'))).exists();
  }

  @Step('Sign in as <username> without password')
  public async signIn(username: string) {
    await this.signInWithPassword(username, '');
  }

  @Step('Sign out')
  public async signOut() {
    await t.click(t.$('#avatar'));
    await t.click(t.link('Sign Out'));
    await t.button('Sign In').exists();
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
      const link = t.link(label, t.within(t.$('[class*=Navigation_base]')));
      const external = row.getCell('external') === 'true';

      await clickAndWaitForPage(link, external);

      const path = row.getCell('route');
      const url = await t.currentURL();
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
    await t.text(`experiment ${id}`).exists();
  }

  @Step('Require page to have <keywords>')
  public async checkPageHas(keywordsTxt: string) {
    const keywords = keywordsTxt.split(', ');
    await checkTextContentFor(keywords, true);
  }

  @Step('Require page to not have <keywords>')
  public async checkPageDoesNotHave(keywordsTxt: string) {
    const keywords = keywordsTxt.split(', ');
    await checkTextContentFor(keywords, false);
  }

  @Step('Navigate to task list page')
  public async navigateToTaskList() {
    await goto(`${BASE_URL}/tasks`);
  }

  @Step('Navigate to master logs page')
  public async navigateToMasterLogs() {
    await goto(`${BASE_URL}/logs`);
  }

  /* Experiment Actions */
  @Step('Activate experiment <id>')
  public async activateExperiment(id: string) {
    await this.navigateToExperimentDetail(id);
    await t.click(t.text('activate'));
    await t.text('pause');
  }

  /* Table Steps */

  @Step('Should have <count> table rows')
  public async checkTableRowCount(count: string) {
    const rows = await getElements('tr[data-row-key]');
    const expectedCount = parseInt(count);
    await assert.strictEqual(rows.length, expectedCount);
  }

  @Step('Sort table by column <column>')
  public async sortTableByColumn(column: string) {
    await t.click(t.text(column));
  }

  @Step('Toggle all table row selection')
  public async toggleAllTableRowSelection() {
    await t.click(t.$('th input[type=checkbox]'));
  }

  @Step('Table batch should have following buttons <table>')
  public async checkTableBatchButton(table: Table) {
    for (var row of table.getTableRows()) {
      const label = row.getCell('table batch buttons');
      const disabled = row.getCell('disabled') === 'true';
      const batchButton = await t.button(label, t.within(t.$('[class*=TableBatch_base]')));
      await batchButton.exists();
      assert.strictEqual(await batchButton.isDisabled(), disabled);
    }
  }

  @Step('<action> all table rows')
  public async actionOnAllExperiments(action: string) {
    await t.click(t.button(action, t.within(t.$('[class*=TableBatch_base]'))));
    // Wait for the modal to animate in
    await t.waitFor(async () => !(await t.$('.ant-modal.zoom-enter').exists()));
    await t.click(t.button(action, t.within(t.$('.ant-modal-body'))));
    // Wait for the modal to animate away
    await t.waitFor(async () => !(await t.$('.ant-modal.zoom-leave').exists()));
  }

  /* Notebook and TensorBoard Steps */

  @Step('Launch notebook')
  public async launchNotebook() {
    await clickAndCloseTab(t.button('Launch Notebook'));
  }

  @Step('Launch cpu-only notebook')
  public async launchCpuNotebook() {
    await t.click(t.$('[class*=NavigationSideBar_launchIcon]'));
    await clickAndCloseTab(t.text('Launch CPU-only Notebook'));
  }

  @Step('Launch tensorboard')
  public async launchTensorboard() {
    await clickAndCloseTab(t.button('View in TensorBoard'));
  }

  /* Dashboard Page Steps */

  @Step('Should have <count> recent task cards')
  public async checkRecentTasks(count: string) {
    const expectedCount = parseInt(count);
    const cards = await getElements('[class*=TaskCard_base]');
    await assert.strictEqual(cards.length, expectedCount);
  }

  /* Experiment List Page Steps */

  @Step('<action> experiment row <row>')
  public async modifyExperiment(action: string, row: string) {
    await t.click(t.tableCell({ row: parseInt(row) + 1, col: 12 }));
    await t.click(t.text(action, t.within(t.$('.ant-dropdown'))));
  }

  @Step('Open TensorBoard from experiment row <row>')
  public async openExperimentInTensorBoard(row: string) {
    await t.click(t.tableCell({ row: parseInt(row) + 1, col: 12 }));
    await clickAndCloseTab(t.text('View in TensorBoard', t.within(t.$('.ant-dropdown'))));
  }

  @Step('Toggle show archived button')
  public async toggleShowArchived() {
    await t.click(t.text('Show'));
  }

  /* Experiment Detail Page Steps */

  @Step('Archive experiment')
  public async archiveExperiment() {
    await t.click(t.button('Archive'));
  }

  @Step('Unarchive experiment')
  public async unarchiveExperiment() {
    await t.click(t.button('Unarchive'));
  }

  @Step('Kill experiment')
  public async killExperiment() {
    await t.click(t.button('Kill'));
    await t.click(t.button('Yes'));
  }

  @Step('View experiment in TensorBoard')
  public async viewExperimentInTensorBoard() {
    await t.click(t.button('View in TensorBoard'));
  }

  /* Task List Page Steps */

  @Step('Filter tasks by type <table>')
  public async filterTasksByType(table: Table) {
    for (var row of table.getTableRows()) {
      const ariaLabel = row.getCell('aria-label');
      const count = row.getCell('count');
      await t.click(t.$(`[aria-label=${ariaLabel}]`));
      await this.checkTableRowCount(count);
      await t.click(t.$(`[aria-label=${ariaLabel}]`));
    }
  }

  /* Cluster */

  @Step('Should show <count> resource pool cards')
  public async checkResourcePoolCardCount(count: string) {
    const cards = await getElements('div[class*=ResourcePoolCard_base]');
    assert.strictEqual(cards.length, parseInt(count));
  }

  @Step('Should show <count> agents in stats')
  public async checkAgentCountStats(count: string) {
    const stats = await getElements('div[class*=OverviewStats_base]');
    assert.strictEqual(stats.length, 3);
    const numAgents = (await stats[0].text()).replace('Connected Agents', '');
    assert.strictEqual(parseInt(numAgents), parseInt(count));
  }

  @Step('Page should contain <text>')
  public async checkTextExist(value: string) {
    assert.ok(t.text(value).exists());
  }

  /* Logs */
  @Step('Should have some log entries')
  public async checkSomeLogLines() {
    const logs = await getElements('[class*=LogViewer_line]');
    assert.ok(logs.length > 0);
  }
}
