import { BaseComponent } from 'playwright-page-model-base/BaseComponent';
import { BaseOverlay, OverlayArgs } from 'playwright-page-model-base/BaseOverlay';

/**
 * Represents the Modal component from antd/es/modal/index.d.ts
 */
export class Modal extends BaseOverlay {
  constructor(args: OverlayArgs) {
    super({
      ...args,
      selector: '.ant-modal-content',
    });
  }
  readonly header = new ModalHeader({ parent: this, selector: '.ant-modal-header' });
  readonly body = new BaseComponent({ parent: this, selector: '.ant-modal-body' });
  readonly footer = new ModalFooter({ parent: this, selector: '.ant-modal-footer' });

  /**
   * Closes the Modal.
   */
  async close(): Promise<void> {
    // Popover has no close button and doesn't respect Escape key
    await this.footer.cancel.pwLocator.click();
    await this.pwLocator.waitFor({ state: 'hidden' });
  }
}

/**
 * Represents the header from the Modal component
 */
class ModalHeader extends BaseComponent {
  readonly title = new BaseComponent({ parent: this, selector: '.ant-modal-title' });
}

/**
 * Represents the footer from the Modal component
 */
class ModalFooter extends BaseComponent {
  readonly submit = new BaseComponent({ parent: this, selector: '[type="submit"]' });
  readonly cancel = new BaseComponent({ parent: this, selector: '[type="cancel"]' });
}
