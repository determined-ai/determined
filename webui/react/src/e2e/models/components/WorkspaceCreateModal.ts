import { BaseComponent } from 'e2e/models/common/base/BaseComponent';
import { Modal } from 'e2e/models/common/hew/Modal';
import { Toggle } from 'e2e/models/common/hew/Toggle';

/**
 * Represents the WorkspaceCreateModal component in src/components/WorkspaceCreateModal.tsx
 */
export class WorkspaceCreateModal extends Modal {
  readonly workspaceName = new BaseComponent({
    parent: this.body,
    selector: 'input[id="workspaceName"]',
  });

  readonly useAgentUser = new Toggle({
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

  readonly useAgentGroup = new Toggle({
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

  readonly useCheckpointStorage = new Toggle({
    parent: this.body,
    selector: '[data-testid="useCheckpointStorage"]',
  });
  // We need more work on this to handle input well most likely since the code editor is complex
  readonly checkpointCodeEditor = new BaseComponent({
    parent: this.body,
    selector: 'div.cm-editor',
  });
}
