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
  readonly workspaceName = new BaseComponent({
    parent: this.body,
    selector: 'input[id="workspaceName"]',
  });

  readonly useAgentUser = new Switch({
    parent: this.body,
    selector: '[data-testid="useAgentUser"]',
  });
  readonly agentUid = new BaseComponent({
    parent: this.body,
    selector: 'input[id="agentUid"]',
  });
  readonly agentUser = new BaseComponent({
    parent: this.body,
    selector: 'input[id="agentUser"]',
  });

  readonly useAgentGroup = new Switch({
    parent: this.body,
    selector: '[data-testid="useAgentGroup"]',
  });
  readonly agentGid = new BaseComponent({
    parent: this.body,
    selector: 'input[id="agentGid"]',
  });
  readonly agentGroup = new BaseComponent({
    parent: this.body,
    selector: 'input[id="agentGroup"]',
  });

  readonly useCheckpointStorage = new Switch({
    parent: this.body,
    selector: '[data-testid="useCheckpointStorage"]',
  });
  // We need more work on this to handle input well most likely since the code editor is complex
  readonly checkpointCodeEditor = new BaseComponent({
    parent: this.body,
    selector: 'div.cm-editor',
  });
}
