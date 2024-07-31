import { BaseOverlay, OverlayArgs } from 'playwright-page-model-base/BaseOverlay';

/**
 * Represents the Popver component from antd/es/popover/index.js
 */
export class Popover extends BaseOverlay {
  constructor(args: OverlayArgs) {
    super({
      ...args,
      selector: '.ant-popover .ant-popover-content .ant-popover-inner-content:visible',
    });
  }

  /**
   * Closes the popover.
   */
  async close(): Promise<void> {
    // [ET-284] Popover click handle doesn't work unless we wait
    await this.root._page.waitForTimeout(500);
    // Popover has no close button and doesn't respect Escape key
    await this.root.nav.sidebar.header.pwLocator.click();
    await this.pwLocator.waitFor({ state: 'hidden' });
  }
}
