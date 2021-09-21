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

const selectors = {
  antdTableRows: '.ant-table-container tbody tr[data-row-key]',
  antdTable: '.ant-table-container tbody tr[data-row-key]',
  antdEmptyTable: '.ant-table-container .ant-empty',
};

const BATCH_ACTION_TEXT = 'Select an action...';
const BATCH_CLEAR_TEXT = 'Clear';

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

const sleep = (ms = 1000) => {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
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
 * A hash id is generated using the content of the element, and the same id is
 * used for deduping. Sometimes the content of the element can change while
 * we attempt to generate a hash id, so an optional `hashSelector` allows us to
 * specifically target a specific content of a given element to be the unique id.
 */
const getElements = async (selector: string, hashSelector?: string): Promise<t.Element[]> => {
  const map: Record<string, boolean> = {};
  const dedupedElements = [];

  try {
    const elements = await t.$(selector).elements();
    for (const element of elements) {
      const hashElement = hashSelector ? await t.$(hashSelector, t.within(element)) : element;
      const hashText = await hashElement.text();
      const hashId = hashText.replace(/\s+/g, ' ').replace(/\r?\n|\r/g, '');

      if (!map[hashId]) {
        map[hashId] = true;
        dedupedElements.push(element);
      }
    }
  } catch (e) {}

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
    await t.waitFor(async () => !(await t.$(selector).exists(undefined, 2000)));
  }

  @Step('Should have element <selector> present')
  public async hasElement(selector: string) {
    await t.waitFor(async () => await t.$(selector).exists());
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
      await t.waitFor(() => url.includes(path));

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
    await t.$(selectors.antdTable).exists();
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
    await t.$(selectors.antdTable).exists();
    // TODO this should check that the table is not in loading state.
    const expectedCount = parseInt(count);
    if (expectedCount === 0) {
      await t.$(selectors.antdEmptyTable).exists();
      return;
    }
    const rows = await getElements(selectors.antdTableRows, '.ant-table-cell:nth-child(2)');
    expect(rows).toHaveLength(expectedCount);
  }

  @Step('Sort table by column <column>')
  public async sortTableByColumn(column: string) {
    await t.click(t.text(column));
  }

  @Step('Toggle all table row selection')
  public async toggleAllTableRowSelection() {
    await t.waitFor(async () => await t.$('th input[type=checkbox]').exists());
    await t.click(t.$('th input[type=checkbox]'));
  }

  @Step('Table batch should have following buttons <table>')
  public async checkTableBatchButton(table: Table) {
    await t.click(t.text(BATCH_ACTION_TEXT));
    for (var row of table.getTableRows()) {
      await t.waitFor(async () => {
        const disabled = row.getCell('disabled') === 'true';
        const label = row.getCell('table batch buttons');
        const menuItem = await t.text(label);
        const menuItemExists = await menuItem.exists();
        const selector = `.ant-select-item${disabled ? '-option-disabled' : ''}`;
        const isValid = await t.text(label, t.within(t.$(selector))).exists();
        return menuItemExists && isValid;
      });
    }
    await t.click(t.button(BATCH_CLEAR_TEXT));
  }

  @Step('<action> all table rows')
  public async actionOnAllTableRows(action: string) {
    await t.click(BATCH_ACTION_TEXT);
    // Wait for the dropdown animation to finish
    await sleep(500);
    await t.click(action, t.within(t.$('.ant-select-dropdown')));
    // Wait for the modal to animate in.
    await t.waitFor(async () => !(await t.$('.ant-modal.zoom-enter').exists()));
    await t.click(t.button(action, t.within(t.$('.ant-modal-body'))));
    // Wait for the modal to animate away
    await t.waitFor(async () => !(await t.$('.ant-modal.zoom-leave').exists()));
  }

  @Step('Scroll table to the <direction>')
  public async scrollTable(direction: string) {
    const tableSelector = '.ant-table-content';
    const scrollAmount = 1000;
    if (direction === 'down') {
      await t.scrollDown(t.$(tableSelector), scrollAmount);
    } else if (direction === 'left') {
      await t.scrollLeft(t.$(tableSelector), scrollAmount);
    } else if (direction === 'right') {
      await t.scrollRight(t.$(tableSelector), scrollAmount);
    } else if (direction === 'up') {
      await t.scrollUp(t.$(tableSelector), scrollAmount);
    }
  }

  @Step('Filter table header <label> with option <option>')
  public async filterTable(label: string, option: string) {
    await t.click(t.$('.ant-table-filter-trigger', t.near(label)));
    await t.click(option, t.within(t.$('.ant-table-filter-dropdown')));
    await t.click(t.button('Ok'), t.within(t.$('.ant-table-filter-dropdown')));
  }

  /* Notebook and TensorBoard Steps */

  //Notebook tests are the same, they both just choose the first resource pool
  @Step('Launch notebook')
  public async launchNotebook() {
    await t.click('Launch JupyterLab');
    // Wait for the modal to animate in
    await t.waitFor(async () => !(await t.$('.ant-modal.zoom-enter').exists()));
    await t.click(t.$('.ant-select-selector'), t.near('Resource Pool'));
    await t.click(t.$('.ant-select-item-option-content'));
    await clickAndCloseTab(t.button('Launch'));
  }

  @Step('Launch cpu-only notebook')
  public async launchCpuNotebook() {
    await t.click(t.button('Launch JupyterLab'));
    // Wait for the modal to animate in
    await t.waitFor(async () => !(await t.$('.ant-modal.zoom-enter').exists()));
    await t.click(t.$('.ant-select-selector'), t.near('Resource Pool'));
    await t.click(t.$('.ant-select-item-option-content'));
    await clickAndCloseTab(t.button('Launch'));
  }

  @Step('Launch tensorboard')
  public async launchTensorboard() {
    await clickAndCloseTab(t.button('View in TensorBoard'));
  }

  /* Dashboard Page Steps */

  @Step('Should have <count> recent task cards')
  public async checkRecentTasks(count: string) {
    await t.waitFor(async () => {
      const expectedCount = parseInt(count);
      const cards = await getElements(
        '[class*=TaskCard_base]',
        '[class*=TaskCard_badges] span:first-of-type',
      );
      return cards.length === expectedCount;
    });
  }

  /* Experiment List Page Steps */

  @Step('<action> experiment row <row>')
  public async modifyExperiment(action: string, row: string) {
    await t.click(t.tableCell({ row: parseInt(row) + 1, col: 13 }));
    await t.click(t.text(action, t.within(t.$('.ant-dropdown'))));
  }

  @Step('Open TensorBoard from experiment row <row>')
  public async openExperimentInTensorBoard(row: string) {
    await t.click(t.tableCell({ row: parseInt(row) + 1, col: 13 }));
    await clickAndCloseTab(t.text('View in TensorBoard', t.within(t.$('.ant-dropdown'))));
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
      await t.click(t.$('.ant-table-thead th:nth-child(3) .ant-table-filter-trigger'));
      await t.click(t.text(ariaLabel, t.within(t.$('.ant-table-filter-dropdown'))));
      await t.click(t.$('[aria-label="Apply Filter"]'));
      await this.checkTableRowCount(count);
      await t.click(t.$('.ant-table-thead th:nth-child(3) .ant-table-filter-trigger'));
      await t.click(t.$('[aria-label="Reset Filter"]'));
    }
  }

  /* Cluster */

  @Step('Should show <count> resource pool cards')
  public async checkResourcePoolCardCount(count: string) {
    await t.waitFor(async () => {
      const cards = await getElements('div[class*=ResourcePoolCard_base]');
      return cards.length === parseInt(count);
    });
  }

  @Step('Should show <count> agents in stats')
  public async checkAgentCountStats(count: string) {
    await t.waitFor(async () => {
      const stats = await getElements('div[class*=OverviewStats_base]');
      const numAgents = (await stats[0].text()).replace('Connected Agents', '');
      return stats.length === 3 && parseInt(numAgents) === parseInt(count);
    });
  }

  @Step('Page should contain <text>')
  public async checkTextExist(value: string) {
    await t.text(value).exists();
  }

  /* Logs */
  @Step('Should have some log entries')
  public async checkSomeLogLines() {
    await t.waitFor(async () => {
      const logs = await getElements('[class*=LogViewer_line]');
      return logs.length > 0;
    });
  }

  /* Dev */
  // use the steps here to test out Taiko behavior on Determined

  @Step('dev')
  public async dev() {
    await t.$('body').exists();
  }
}
