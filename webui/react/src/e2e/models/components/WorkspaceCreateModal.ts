import { Switch } from 'e2e/models/ant/Switch';
import { BaseComponent } from 'e2e/models/BaseComponent';
import { Modal } from 'e2e/models/hew/Modal';

/**
 * Returns a representation of the Workspace create/edit modal component.
 * This constructor represents the contents in src/components/Page.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Page
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class WorkspaceCreateModal extends Modal {
  readonly workspaceName: BaseComponent = new BaseComponent({
    parent: this.body,
    selector: 'input[id="workspaceName"]',
  });

  readonly useAgentUser: Switch = new Switch({
    parent: this.body,
    selector: '[data-testid="useAgentUser"]',
  });
  readonly agentUid: BaseComponent = new BaseComponent({
    parent: this.body,
    selector: 'input[id="agentUid"]',
  });
  readonly agentUser: BaseComponent = new BaseComponent({
    parent: this.body,
    selector: 'input[id="agentUser"]',
  });

  readonly useAgentGroup: Switch = new Switch({
    parent: this.body,
    selector: '[data-testid="useAgentGroup"]',
  });
  readonly agentGid: BaseComponent = new BaseComponent({
    parent: this.body,
    selector: 'input[id="agentGid"]',
  });
  readonly agentGroup: BaseComponent = new BaseComponent({
    parent: this.body,
    selector: 'input[id="agentGroup"]',
  });

  readonly useCheckpointStorage: Switch = new Switch({
    parent: this.body,
    selector: '[data-testid="useCheckpointStorage"]',
  });
  // We need more work on this to handle input well most likely since the code editor is complex
  readonly checkpointCodeEditor: BaseComponent = new BaseComponent({
    parent: this.body,
    selector: 'div.cm-editor',
  });
}
