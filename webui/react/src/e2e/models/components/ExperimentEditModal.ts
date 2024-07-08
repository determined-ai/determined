import { BaseComponent } from 'e2e/models/common/base/BaseComponent';
import { Modal } from 'e2e/models/common/hew/Modal';

export default class ExperimentEditModal extends Modal {
  readonly nameInput = new BaseComponent({
    parent: this,
    selector: 'input[id="experimentName"]',
  });
}
