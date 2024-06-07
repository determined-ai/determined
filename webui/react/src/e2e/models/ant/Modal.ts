import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Represents the Modal component from antd/es/modal/index.d.ts
 */
export class Modal extends NamedComponent {
  readonly defaultSelector = '.ant-modal-content';
  readonly header = new ModalHeader({ parent: this, selector: '.ant-modal-header' });
  readonly body = new BaseComponent({ parent: this, selector: '.ant-modal-body' });
  readonly footer = new ModalFooter({ parent: this, selector: '.ant-modal-footer' });
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
