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
import { setViewPort } from 'taiko';

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
const viewports = {
  mobile: { width: 411, height: 823 },
  desktop: { width: 960, height: 1536 },
};

function handleError(e) {
  console.error(e);
}

export default class StepImplementation {
  @BeforeSuite()
  public async beforeSuite() {
    try {
      const browserArgs = HEADLESS
        ? { headless: true, args: ['--no-sandbox', '--disable-setuid-sandbox'] }
        : { headless: false };
      await openBrowser(browserArgs);
    } catch (e) {
      handleError(e);
    }
  }

  @AfterSuite()
  public async afterSuite() {
    try {
      await closeBrowser();
    } catch (e) {
      handleError(e);
    }
  }

  @BeforeScenario()
  public async startScreencast(context: ExecutionContext) {
    try {
      if (SCREENCAST) {
        const spec = context.getCurrentSpec().getName();
        const scenario = context.getCurrentScenario().getName();
        const filename = `screencasts/${spec} -- ${scenario}.gif`;
        await screencast.startScreencast(filename);
      }
    } catch (e) {
      handleError(e);
    }
  }

  @AfterScenario()
  public async stopScreencast() {
    try {
      if (SCREENCAST) {
        await screencast.stopScreencast();
      }
    } catch (e) {
      handleError(e);
    }
  }

  /* Authentication Steps */

  @Step('Sign in as <username> with <password>')
  public async signInWithPassword(username: string, password: string) {
    try {
      await goto(`${BASE_URL}/login`);
      await clear(focus(textBox('username')));
      await write(username);
      if (password !== '') {
        await focus(textBox('password'));
        await write(password);
      }
      await click(button('Sign In'));
      await text(username, near($('#avatar'))).exists();
    } catch (e) {
      handleError(e);
    }
  }

  @Step('Sign in as <username> without password')
  public async signIn(username: string) {
    try {
      await this.signInWithPassword(username, '');
    } catch (e) {
      handleError(e);
    }
  }

  @Step('Sign out')
  public async signOut() {
    try {
      await click($('#avatar'));
      await click(link('Sign Out'));
      await button('Sign In').exists();
    } catch (e) {
      handleError(e);
    }
  }

  /* Browser Utility Steps */

  @Step('Should not have element <selector> present')
  public async hasNoElement(selector: string) {
    const exists = await $(selector).exists(1000);
    assert.ok(!exists);
  }
  @Step('Should have element <selector> present')
  public async hasElement(selector: string) {
    await $(selector).exists();
  }

  @Step('Switch to mobile view')
  public async setMobileViewport() {
    await setViewPort(viewports.mobile);
  }

  @Step('Switch to desktop view')
  public async setDesktopViewport() {
    await setViewPort(viewports.desktop);
  }

  /* Navigation Steps */

  @Step('Navigate to the following routes <table>')
  public async navigateWithTable(table: Table) {
    try {
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
    } catch (e) {
      handleError(e);
    }
  }

  @Step('Navigate to dashboard page')
  public async navigateToDashboard() {
    try {
      await goto(`${BASE_URL}/dashboard`);
    } catch (e) {
      handleError(e);
    }
  }

  @Step('Navigate to experiment list page')
  public async navigateToExperimentList() {
    try {
      await goto(`${BASE_URL}/experiments`);
    } catch (e) {
      handleError(e);
    }
  }

  @Step('Navigate to experiment <id> page')
  public async navigateToExperimentDetail(id: string) {
    try {
      await goto(`${BASE_URL}/experiments/${id}`);
    } catch (e) {
      handleError(e);
    }
  }

  @Step('Navigate to task list page')
  public async navigateToTaskList() {
    try {
      await goto(`${BASE_URL}/tasks`);
    } catch (e) {
      handleError(e);
    }
  }

  @Step('Navigate to master logs page')
  public async navigateToMasterLogs() {
    try {
      await goto(`${BASE_URL}/logs`);
    } catch (e) {
      handleError(e);
    }
  }

  /* Table Steps */

  @Step('Should have <count> table rows')
  public async checkTableRowCount(count: string) {
    try {
      const expectedCount = parseInt(count);
      await assert.strictEqual((await $('tr[data-row-key]').elements()).length, expectedCount);
    } catch (e) {
      handleError(e);
    }
  }

  @Step('Sort table by column <column>')
  public async sortTableByColumn(column: string) {
    try {
      await click(text(column));
    } catch (e) {
      handleError(e);
    }
  }

  @Step('Select all table rows')
  public async selectAllTableRows() {
    try {
      await click($('th input[type=checkbox]'));
    } catch (e) {
      handleError(e);
    }
  }

  @Step('Table batch should have following buttons <table>')
  public async checkTableBatchButton(table: Table) {
    try {
      for (var row of table.getTableRows()) {
        const label = row.getCell('table batch buttons');
        const disabled = row.getCell('disabled') === 'true';
        const batchButton = await button(label, within($('[class*=TableBatch_base]')));
        await batchButton.exists();
        assert.strictEqual(await batchButton.isDisabled(), disabled);
      }
    } catch (e) {
      handleError(e);
    }
  }

  @Step('<action> all table rows')
  public async actionOnAllExperiments(action: string) {
    try {
      await click(button(action, within($('[class*=TableBatch_base]'))));
      // Wait for the modal to animate in
      await waitFor(async () => !(await $('.ant-modal.zoom-enter').exists()));
      await click(button(action, within($('.ant-modal-body'))));
      // Wait for the modal to animate away
      await waitFor(async () => !(await $('.ant-modal.zoom-leave').exists()));
    } catch (e) {
      handleError(e);
    }
  }

  /* Notebook and TensorBoard Steps */

  @Step('Launch notebook')
  public async launchNotebook() {
    try {
      await click(button('Launch Notebook'), { waitForEvents: ['targetNavigated'] });
      await /http(.*)(wait|proxy)/;
    } catch (e) {
      handleError(e);
    }
  }

  @Step('Launch cpu-only notebook')
  public async launchCpuNotebook() {
    try {
      await click($('[class*=Navigation_launchIcon]'));
      await click(text('Launch CPU-only Notebook'));
    } catch (e) {
      handleError(e);
    }
  }

  @Step('Launch tensorboard')
  public async launchTensorboard() {
    try {
      await click(button('View in TensorBoard'), { waitForEvents: ['targetNavigated'] });
    } catch (e) {
      handleError(e);
    }
  }

  @Step('Close current tab')
  public async closeWaitPage() {
    try {
      await closeTab();
    } catch (e) {
      handleError(e);
    }
  }

  /* Dashboard Page Steps */

  @Step('Should have <count> recent task cards')
  public async checkRecentTasks(count: string) {
    try {
      const expectedCount = parseInt(count);
      await assert.strictEqual(
        (await $('[class*=TaskCard_base]').elements()).length,
        expectedCount,
      );
    } catch (e) {
      handleError(e);
    }
  }

  /* Experiment List Page Steps */

  @Step('<action> experiment row <row>')
  public async modifyExperiment(action: string, row: string) {
    try {
      await click(tableCell({ row: parseInt(row) + 1, col: 11 }));
      await click(text(action, within($('.ant-dropdown'))));
    } catch (e) {
      handleError(e);
    }
  }

  @Step('Toggle show archived button')
  public async toggleShowArchived() {
    try {
      await click(button({ class: 'ant-switch' }));
    } catch (e) {
      handleError(e);
    }
  }

  /* Task List Page Steps */

  @Step('Filter tasks by type <table>')
  public async filterTasksByType(table: Table) {
    try {
      for (var row of table.getTableRows()) {
        const ariaLabel = row.getCell('aria-label');
        const count = row.getCell('count');
        await click($(`[aria-label=${ariaLabel}]`));
        await this.checkTableRowCount(count);
        await click($(`[aria-label=${ariaLabel}]`));
      }
    } catch (e) {
      handleError(e);
    }
  }
}
