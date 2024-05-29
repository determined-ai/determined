import { BaseComponent } from 'e2e/models/BaseComponent';
import { Modal } from 'e2e/models/hew/Modal';

/**
 * Returns a representation of the Project create/edit modal component.
 * This constructor represents the contents in src/components/Page.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The model's parent in the page hierarchy
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class ProjectCreateModal extends Modal {
  readonly projectName = new BaseComponent({
    parent: this,
    selector: 'input[id="projectName"]',
  });

  readonly description = new BaseComponent({
    parent: this,
    selector: 'input[id="description"]',
  });
}
