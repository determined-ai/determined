import { BaseComponent } from 'e2e/models/BaseComponent';
import { Modal } from 'e2e/models/hew/Modal';

/**
 * Returns a representation of the Workspace create/edit modal component.
 * This constructor represents the contents in src/components/Page.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Page
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class ProjectCreateModal extends Modal {
  readonly projectName: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'input[id="projectName"]',
  });

  readonly description: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'input[id="description"]',
  });
}
