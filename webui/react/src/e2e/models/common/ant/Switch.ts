import { expect } from 'e2e/fixtures/global-fixtures';
import { NamedComponent } from 'e2e/models/common/base/BaseComponent';

/**
 * Represents the Switch component from antd/es/switch/index.js
 */
export class Switch extends NamedComponent {
  defaultSelector = 'button[role="switch"]';
  private async isChecked(): Promise<boolean> {
    return await this.pwLocator.getAttribute('aria-checked').then((value) => value === 'true');
  }
  async check(): Promise<void> {
    if (!(await this.isChecked())) {
      await this.pwLocator.click();
      await expect.poll(async () => await this.isChecked()).toBeTruthy();
    }
  }
  async uncheck(): Promise<void> {
    if (await this.isChecked()) {
      await this.pwLocator.click();
      await expect.poll(async () => await this.isChecked()).toBeFalsy();
    }
  }
}
